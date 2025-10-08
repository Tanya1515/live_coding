package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"sync"
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

Необходимо реализовать структуру CombinedStream, которая объединяет
несколько объектов, реализующих интерфейс MeasuredStream.
Структура CombinedStream должна сама реализовывать интерфейс MeasuredStream.

Операции, которые должен поддерживать CombinedStream:

Read:  должен читать данные последовательно из всех переданных потоков в том
             же порядке, в котором они переданы в NewCombinedStream
Seek:  должен позволять перемещать указатель на заданную позицию в объединенной
            последовательности потоков.
Close: должен закрыть все потоки.
Size:  должен возвращать суммарный размер данных всех потоков.

Здесь whence - аргумент, который определяет позицию относительно некоторой стартовой точки:
1) io.SeekStart - позиция выставялется относительно начала потока
2) io.SeekCurrent - позиция выставляется относительно текущей позиции
3) io.SeekEnd - позиция выставялется относительно конца потока

*/

const (
	seekStart = iota
	seekCurrent
	seekEnd
)
const bufferSize = 1024 * 1024
type MeasuredStream interface {
	io.ReadSeekCloser
	TotalSize() int64
}

type CombinedStream struct {
	streams        []MeasuredStream
	buffer         []byte
	bufferPointer  int64
	totalSize      int64
	currentPointer int64 // будет передвигаться в трем местах: Read, Seek и processBuffer
	indexStream    int64
	cond           *sync.Cond
	stopChan       chan struct{}
}

func (cs *CombinedStream) processBuffer() {
	cs.cond.L.Lock()
	defer cs.cond.L.Unlock()
	for {
		cs.cond.Wait()
		select {
		case <-cs.stopChan:
			return
		default:
		}

		processed := 0
		for cs.indexStream < int64(len(cs.streams)) {
			n, err := cs.streams[cs.indexStream].Read(cs.buffer[processed:])
			processed += n
			cs.currentPointer += int64(n)
			if len(cs.buffer) == bufferSize {
				break
			}
			if err == io.EOF {
				cs.indexStream++
				continue
			}
			if err != nil {
				log.Println("error while reading from channel")
			}
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
	mu := sync.Mutex{}
	cond := sync.NewCond(&mu)
	cs := &CombinedStream{
		streams:   rs,
		buffer:    buffer,
		totalSize: size,
		stopChan:  stopChan,
		cond:      cond,
	}

	go cs.processBuffer()
	cs.cond.Signal()

	return cs
}

func (cs *CombinedStream) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, fmt.Errorf("buffer for reading is Empty")
	}
	if cs.TotalSize() == 0 {
		return 0, io.EOF
	}
	processedSize := 0
	cs.cond.L.Lock()
	defer cs.cond.L.Unlock()
	processedSize += copy(p, cs.buffer[cs.bufferPointer:])
	cs.bufferPointer += int64(processedSize)
	defer func() {
		ok := false
		cs.cond.L.Lock()
		if cs.bufferPointer == bufferSize {
			ok = true
		}
		cs.cond.L.Unlock()
		if ok {
			cs.cond.Signal()
		}
	}()
	for {
		if cs.indexStream == int64(len(cs.streams)) {
			break
		}
		if processedSize == len(p) {
			return processedSize, nil
		}

		n, err := cs.streams[cs.indexStream].Read(p[processedSize:])
		processedSize += n
		cs.currentPointer += int64(n)
		if err != nil {
			if err == io.EOF {
				cs.indexStream++
				continue
			}
			return processedSize, fmt.Errorf("error while reading data from stream: %w", err)
		}
	}
	return processedSize, io.EOF
}

func (cs *CombinedStream) Seek(offset int64, whence int) (int64, error) {
	if cs.totalSize == 0 {
		if offset == 0 && (whence == io.SeekStart || whence == io.SeekEnd) {
			return 0, nil
		}

		return 0, io.EOF
	}

	cs.cond.L.Lock()
	bufferSize := bufferSize - cs.bufferPointer + 1
	var absPos int64
	switch whence {
	case seekStart:
		absPos = offset
	case seekCurrent:
		absPos = cs.currentPointer - bufferSize + offset
	case seekEnd:
		absPos = cs.totalSize + offset
	default:
		return 0, fmt.Errorf("error: invalid whence")
	}
	if absPos < 0 || absPos > cs.totalSize {
		return 0, fmt.Errorf("invalid offset")
	}
	var streamIndex int
	var sumStreams int64
	for {
		if sumStreams > absPos {
			break
		}
		sumStreams += cs.streams[streamIndex].TotalSize()
		streamIndex++
	}
	sumStreams -= cs.streams[streamIndex].TotalSize()
	localOffset := absPos - sumStreams

	for i := 0; i < streamIndex; i++ {
		_, err := cs.streams[i].Seek(0, seekEnd)
		if err != nil {
			return 0, fmt.Errorf("error while seeking stream")
		}
	}
	_, err := cs.streams[streamIndex].Seek(localOffset, seekStart)
	if err != nil {
		return 0, fmt.Errorf("error while seeking stream")
	}
	for i := streamIndex + 1; i < len(cs.streams); i++ {
		_, err = cs.streams[streamIndex].Seek(0, seekStart)
		if err != nil {
			return 0, fmt.Errorf("error while seeking stream")
		}
	}
	cs.currentPointer = absPos
	cs.indexStream = int64(streamIndex)
	cs.cond.L.Unlock()
	cs.cond.Signal()
	return absPos, nil

}

func (cs *CombinedStream) Close() error {
	var resultError error
	cs.cond.Signal()
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
