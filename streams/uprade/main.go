package main

import (
	"errors"
	"io"
	"sync"
	"time"
)

/*

Продвинутый уровень:

Необходимо в CombinedStream добавить функционал для асинхронного префетча
данных в буфер ради ускорения последующих операций чтения.

Изменения будут следующими:
const bufferSize = 1024 * 1024

func NewCombinedStream(buffersNum int, rs ... MeasuredStream) * CombinedStream {
    // put your code here...
    return nil
}

Read: должен возвращать данные всегда из своего внутреннего буфера.
Буфер, в свою очередь, должен в последовательном порядке асинхронно пополняться данными из ридеров.

Seek: должен в первую очередь перемещать курсор на позицию внутри буфера.
В случае, если позиция находится за пределами буфера префетча, тогда переходить
на необходимую позицию в объединенной последовательности ридеров.

Close: должен обеспечивать корректное закрытие и освобождение всех ресурсов.

*/

const (
	seekStart = iota
	seekCurrent
	seekEnd
)

type MeasuredStream interface {
	io.ReadSeekCloser
	TotalSize() int64
}

type CombinedStream struct {
	streams            []MeasuredStream
	buffer             []byte
	mu                 *sync.Mutex
	bufferPointer      int
	bufferPointerWrite int
	totalSize          int64
	currentPointer     int64
	indexStream        int
	cond               *sync.Cond
	stopChan           chan struct{}
}

const bufferSize = 1024 * 1024

func (cs *CombinedStream) processBuffer() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-cs.stopChan:
			return
		case <-ticker.C:
            cs.mu.Lock()
            var diff int
            if cs.bufferPointerWrite > cs.bufferPointer {
                diff = cs.bufferPointerWrite - cs.bufferPointer 
            } else {
                diff = bufferSize - cs.bufferPointer
                diff += cs.bufferPointerWrite
            }

            i := cs.indexStream
            amountRead := 0
            for i < len(cs.streams) {
                if diff >= 1024 {
                    break
                }
                if cs.bufferPointerWrite > cs.bufferPointer {
                    count, err:= cs.streams[i].Read(cs.buffer[cs.bufferPointerWrite:])
                    if err == io.EOF {
                        i += 1
                    }
                    cs.bufferPointerWrite += count
                    diff += count
                    amountRead += count
                    if cs.bufferPointerWrite == bufferSize - 1 {
                        cs.bufferPointerWrite = 0
                    }
                    if err != nil && err != io.EOF {
                        break
                    }
                } else {
                    count, err := cs.streams[i].Read(cs.buffer[cs.bufferPointerWrite:cs.bufferPointer])
                    if err == io.EOF {
                        i += 1
                    }
                    cs.bufferPointerWrite += count
                    diff += count
                    amountRead += count
                    if err != nil && err != io.EOF {
                        break
                    }
                }
            }

            cs.indexStream = i
            cs.currentPointer += int64(amountRead)
            cs.mu.Unlock()
		}
	}
}

func NewCombinedStream(buffersNum int, rs ...MeasuredStream) *CombinedStream {
	var size int64
	stopChan := make(chan struct{})
	buffer := make([]byte, bufferSize)

	for _, stream := range rs {
		size += stream.TotalSize()
	}

	return &CombinedStream{
		streams:   rs,
		buffer:    buffer,
		totalSize: size,
		stopChan:  stopChan,
		mu:        &sync.Mutex{},
	}
}

func (cs *CombinedStream) Read(p []byte) (n int, err error) {
	processedSize := 0
	readSize := len(p)
	
	for {
        cs.mu.Lock()
		copiedCount := copy(p[processedSize:], cs.buffer[cs.bufferPointer:cs.bufferPointerWrite])
		cs.bufferPointer += copiedCount
        if cs.bufferPointer == bufferSize - 1 {
            cs.bufferPointer = 0
        }
        if cs.indexStream == len(cs.streams) - 1 {
            cs.mu.Unlock()
            return processedSize, io.EOF
        }
        cs.mu.Unlock()
		processedSize += copiedCount
		if processedSize == readSize {
			break
		}
        time.Sleep(3 * time.Second)
	}
	

	return processedSize, nil
}

func (cs *CombinedStream) Seek(offset int64, whence int) (int64, error) {
    switch whence {
    case seekStart: 
        cs.mu.Lock()
        cs.bufferPointer = 0
        cs.bufferPointerWrite = 0
        cs.currentPointer = 0
        for i := 0; i < len(cs.streams); i++ {
            if offset <= bufferSize {
                count, _ := cs.streams[i].Read(cs.buffer[cs.bufferPointerWrite:])
                cs.bufferPointerWrite += count
            }
            streamSize := cs.streams[i].TotalSize()
			if streamSize > offset {
                cs.currentPointer += offset
				cs.indexStream = i
				cs.streams[i].Seek(offset, seekCurrent)
				break
			} else {
				offset -= streamSize
				cs.streams[i].Seek(streamSize, seekStart)
                cs.currentPointer += streamSize
			}
        }
        cs.mu.Unlock()
    case seekCurrent:
        cs.mu.Lock()
        cs.bufferPointer = 0
        cs.bufferPointerWrite = 0
        var size int64
        for i := 0; i <= cs.indexStream; i++ {
            size += cs.streams[i].TotalSize()
        }
        currentOffset := size - cs.currentPointer
        streamSize := cs.streams[cs.indexStream].TotalSize() - currentOffset
        for i := cs.indexStream; i < len(cs.streams); i++ {
            if offset <= bufferSize {
                count, _ := cs.streams[i].Read(cs.buffer[cs.bufferPointerWrite:])
                cs.bufferPointerWrite += count
            }
            if streamSize > offset {
                cs.currentPointer += offset
				cs.indexStream = i
				cs.streams[i].Seek(offset, seekCurrent)
				break
			} else {
				offset -= streamSize
				cs.streams[i].Seek(streamSize, seekStart)
                cs.currentPointer += streamSize
			}
            streamSize = cs.streams[i].TotalSize()
        }
        cs.mu.Unlock()
    case seekEnd:
        cs.mu.Lock()
        cs.bufferPointer = 0
        cs.bufferPointerWrite = 0
        cs.currentPointer = cs.TotalSize()
        for i := len(cs.streams) - 1; i >= 0; i-- { 
            streamSize := cs.streams[i].TotalSize()
            currentOffset := offset * (-1)
            if currentOffset <= bufferSize {
                // проблема: читаем не с конца
                count, _ := cs.streams[i].Read(cs.buffer[cs.bufferPointerWrite:])
                cs.bufferPointerWrite += count
            }
            if streamSize > offset {
				cs.indexStream = i
                cs.currentPointer += offset 
				cs.streams[i].Seek(offset, seekEnd)
				break
			} else {
				offset += streamSize
				cs.streams[i].Seek(streamSize, seekStart)
				cs.currentPointer -= streamSize
			}
        }

        cs.mu.Unlock()
    }

    if offset > 0 {
        return -1, io.EOF
    }

    return cs.currentPointer, nil
}

func (cs *CombinedStream) Close() error {
	var resultError error
	close(cs.stopChan)
	for _, stream := range cs.streams {
		errCur := stream.Close()
		resultError = errors.Join(resultError, errCur)
	}
	return resultError
}

func (cs *CombinedStream) TotalSize() int64 {
	return cs.totalSize

}
