package main

import (
	"errors"
	"fmt"
	"io"
	"sync"
)

const bufferSize = 1024 * 1024

type MeasuredStream interface {
	io.ReadSeekCloser
	TotalSize() int64
}

// Структура для хранения буфера
type BufferItem struct {
	start int64
	data  []byte
}

type CombinedStream struct {
	streams     []MeasuredStream // все соединенные потоки
	queue       []*BufferItem    // очередь префетченных буферов
	currentBuf  *BufferItem      // текущий буфер для чтения
	bufPointer  int              // позиция в currentBuf.data
	totalSize   int64
	currentPos  int64 // глобальная позиция
	indexStream int   // текущий индекс stream-а для префетча
	prefetchPos int64 // следующая позиция для префетча
	done        bool
	err         error      // ошибка
	cond        *sync.Cond // для синхронизации
	stopChan    chan struct{}
}

func NewCombinedStream(buffersNum int, rs ...MeasuredStream) *CombinedStream {
	var size int64
	stopChan := make(chan struct{})

	// считаем сумму всех stream-ов
	for _, stream := range rs {
		size += stream.TotalSize()
	}
	mu := sync.Mutex{}
	cond := sync.NewCond(&mu)
	cs := &CombinedStream{
		streams:   rs,
		queue:     make([]*BufferItem, 0, buffersNum),
		totalSize: size,
		stopChan:  stopChan,
		cond:      cond,
	}
	// Инициализация позиций в стримах
	cs.resetSeeks(0, 0)
	// запускаем горутину для асихронного prefetch-а
	go cs.processBuffer(buffersNum)
	return cs
}

// функция, которая смещает указатель в stream-ах относительно текущего индекса
// если i < streamIndex - сдвигаем в конец
// если i = streamIndex - сдвигаем на текущий localOffset
// если i > streamIndex - сдвигаем в конец
// Таким образом, выравниваем все позиции
func (cs *CombinedStream) resetSeeks(streamIndex int, localOffset int64) error {
	for i := 0; i < len(cs.streams); i++ {
		var off int64
		var wh int
		if i < streamIndex {
			off = 0
			wh = io.SeekEnd
		} else if i == streamIndex {
			off = localOffset
			wh = io.SeekStart
		} else {
			off = 0
			wh = io.SeekStart
		}
		_, err := cs.streams[i].Seek(off, wh)
		if err != nil {
			return err
		}
	}
	return nil
}

func (cs *CombinedStream) processBuffer(buffersNum int) {
	cs.cond.L.Lock()
	defer cs.cond.L.Unlock()
	for {
		for len(cs.queue) < buffersNum && !cs.done && cs.err == nil {
			buf := make([]byte, bufferSize)
			processed := 0
			// пока буфер не заполнен до конца и пока не закончили читать все потоки
			// и пока не встретили ошибки
			for processed < bufferSize && cs.indexStream < len(cs.streams) && cs.err == nil {
				n, err := cs.streams[cs.indexStream].Read(buf[processed:])
				processed += n
				if err != nil {
					if err == io.EOF {
						cs.indexStream++
						continue
					}
					cs.err = err
				}
			}
			// если возникла ошибка при чтении stream-ов - завершаем горутину.
			if cs.err != nil {
				return
			}
			// если что-то записано в буфере, то добавляем записанные
			// данные в очередь буферов
			if processed > 0 {
				item := &BufferItem{
					start: cs.prefetchPos,
					data:  buf[:processed],
				}
				cs.queue = append(cs.queue, item)
				cs.prefetchPos += int64(processed)
			}
			// если индекс текущего stream-а больше или равен количеству всех stream-ов
			// то выставляем флаг, что все данные прочитаны и оповещаем все остальные горутины
			if processed < bufferSize && cs.indexStream >= len(cs.streams) {
				cs.done = true
			}
		}
		select {
		case <-cs.stopChan:
			return
		default:
		}
		// ждём сигнал, что queue full или сигнал от Seek/Read и затем проверяем,
		// что количество буферов не превосходит ранее заданной buffersNum
		// и сам канал не закончился
		cs.cond.Wait()
	}
}

func (cs *CombinedStream) Read(p []byte) (n int, err error) {
	// проверяем, что размер буфера не нулевой
	if len(p) == 0 {
		return 0, nil
	}

	// проверяем, что размер всех каналов не нулевой
	if cs.totalSize == 0 {
		return 0, io.EOF
	}

	cs.cond.L.Lock()
	defer cs.cond.L.Unlock()

	processed := 0
	for processed < len(p) {
		// проверяем что текущий буфер не выставлен или указатель на текущем
		// буфере превосходит размер данных в текущем буфере
		if cs.currentBuf == nil || cs.bufPointer >= len(cs.currentBuf.data) {
			if len(cs.queue) > 0 {
				// Выставляем текущий буфер
				cs.currentBuf = cs.queue[0]
				// Обрезаем очередь буферов
				cs.queue = cs.queue[1:]
				// Выставляем указатель6 откуда читать из текущего буфера на 0-ую позицию
				// (то есть будем читать сначала)
				cs.bufPointer = 0
				cs.cond.Signal() // сигналим, что место в queue освободилось
			} else {
				// Нет префетча — читаем напрямую из стримов
				// Проверяем, что еще есть, что читать из stream-ов (то есть они не закончились)
				if cs.done || cs.indexStream >= len(cs.streams) {
					if processed == 0 {
						return 0, io.EOF
					}
					return processed, io.EOF
				}
				// проверяем, что в stream-ах нет ошибки
				if cs.err != nil {
					return processed, cs.err
				}
				// считываем данные из текущего stream-а
				nDirect, errDirect := cs.streams[cs.indexStream].Read(p[processed:])
				// сдвигаем processed и указатель на прочитанное количество байт
				processed += nDirect
				cs.currentPos += int64(nDirect)
				// проверяем, что не было ошибки при считывании из sstream-ов
				if errDirect != nil {
					// если stream просто закончился - увеличиваем счетчик текущих stream-ов
					if errDirect == io.EOF {
						cs.indexStream++
						continue
					}
					// если реальная ошибка - записываем ее и выходим из операции Read
					cs.err = errDirect
					return processed, fmt.Errorf("error while reading data from stream: %w", errDirect)
				}
			}
		}
		// выбираем минимальное количество байт: которое либо можно скопировать из
		// буфера, либо оставшееся количество байт в stream-е
		copyAmt := min(len(p)-processed, len(cs.currentBuf.data)-cs.bufPointer)
		// копируем данные от текущего указателя до copyAmt
		copy(p[processed:processed+copyAmt], cs.currentBuf.data[cs.bufPointer:cs.bufPointer+copyAmt])
		// увеличиваем указатели на текущее количество байт в буфере, в смерженных stream-ах,
		// в буфере, куда необходимо скопировать данные
		processed += copyAmt
		cs.bufPointer += copyAmt
		cs.currentPos += int64(copyAmt)
	}
	return processed, nil
}

func (cs *CombinedStream) Seek(offset int64, whence int) (int64, error) {
	cs.cond.L.Lock()
	defer cs.cond.L.Unlock()
	var absPos int64

	// Проверяем, что пришла корректная стартовая позиция и
	// относительно этой позиции вычисляем итоговую позицию в stream-е
	switch whence {
	case io.SeekStart:
		absPos = offset
	case io.SeekCurrent:
		absPos = cs.currentPos + offset
	case io.SeekEnd:
		absPos = cs.totalSize + offset
	default:
		return 0, fmt.Errorf("error: invalid whence")
	}

	// проверяем, что итоговая позиция находится в корректном интервале
	if absPos < 0 || absPos > cs.totalSize {
		return 0, fmt.Errorf("invalid offset")
	}
	// проверяем общий размер stream-ов
	if cs.totalSize == 0 {
		cs.currentPos = 0
		return 0, nil
	}

	// Проверка, что итоговая позиция попадает в текущий буфер
	if cs.currentBuf != nil && cs.currentBuf.start <= absPos && absPos < cs.currentBuf.start+int64(len(cs.currentBuf.data)) {
		cs.bufPointer = int(absPos - cs.currentBuf.start)
		cs.currentPos = absPos
		cs.cond.Signal()
		return absPos, nil
	}

	// Проверка в queue
	for i, item := range cs.queue {
		// absPos попадает в интервал конкретного буфера, првоеряем item.start,
		// который является позицией для следующего prefetch
		if item.start <= absPos && absPos < item.start+int64(len(item.data)) {
			cs.currentBuf = item
			cs.bufPointer = int(absPos - item.start)
			cs.queue = cs.queue[i+1:] // обрезаем очередь буферов
			cs.currentPos = absPos
			cs.cond.Signal() // место освободилось
			return absPos, nil
		}
	}

	// Если в буфере места недостаточно - выходим и начинаем итерироваться по оставшимся stream-ам
	cs.queue = cs.queue[:0]
	cs.currentBuf = nil
	cs.bufPointer = 0
	var sumPrev int64
	cs.indexStream = 0
	for cs.indexStream < len(cs.streams) {
		if sumPrev+cs.streams[cs.indexStream].TotalSize() > absPos {
			break
		}
		sumPrev += cs.streams[cs.indexStream].TotalSize()
		cs.indexStream++
	}

	localOffset := absPos - sumPrev

	err := cs.resetSeeks(cs.indexStream, localOffset)
	if err != nil {
		return 0, fmt.Errorf("error while seeking stream: %w", err)
	}

	cs.prefetchPos = absPos
	cs.done = false
	cs.err = nil
	cs.cond.Signal() // перезапускаем префетч
	cs.currentPos = absPos
	return absPos, nil
}

func (cs *CombinedStream) Close() error {
	var resultError error
	cs.cond.L.Lock()
	cs.cond.Broadcast()
	cs.cond.L.Unlock()
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
