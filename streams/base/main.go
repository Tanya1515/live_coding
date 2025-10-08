package main

import (
	"errors"
	"fmt"
	"io"
)

/*
 * Необходимо реализовать структуру CombinedStream, которая объединяет
 * несколько объектов, реализующих интерфейс MeasuredStream.
 * Структура CombinedStream должна сама реализовывать интерфейс MeasuredStream.
 *
 * Операции, которые должен поддерживать CombinedStream:
 *
 * * Read:  должен читать данные последовательно из всех переданных потоков в том
 *             же порядке, в котором они переданы в NewCombinedStream
 * * Seek:  должен позволять перемещать указатель на заданную позицию в объединенной
 *             последовательности потоков.
 * * Close: должен закрыть все потоки.
 * * Size:  должен возвращать суммарный размер данных всех потоков.
 * *
 * * Здесь whence - аргумент, который определяет позицию относительно некоторой стартовой точки:
 * * 1) io.SeekStart - позиция выставялется относительно начала потока
 * * 2) io.SeekCurrent - позиция выставляется относительно текущей позиции
 * * 3) io.SeekEnd - позиция выставялется относительно конца потока
 */

const (
	seekStart = iota
	seekCurrent
	seekEnd
)

type ReadSeekCloser interface {
	Read(p []byte) (n int, err error)
	Seek(offset int64, whence int) (int64, error)
	Close() error
}

type MeasuredStream interface {
	ReadSeekCloser
	TotalSize() int64
}

type CombinedStream struct {
	streams            []MeasuredStream
	indexCurrentStream int
	currentPointer     int64
	totalSize          int64
}

func NewCombinedStream(streams ...MeasuredStream) *CombinedStream {
	var size int64
	for _, stream := range streams {
		size += stream.TotalSize()
	}
	return &CombinedStream{
		streams:   streams,
		totalSize: size,
	}
}

func (cs *CombinedStream) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, fmt.Errorf("buffer for reading is Empty")
	}
	if cs.TotalSize() == 0 {
		return 0, io.EOF
	}
	processedSize := 0

	for i := cs.indexCurrentStream; i < len(cs.streams); i++ {
		n, err := cs.streams[i].Read(p[processedSize : ])
		processedSize += n
		cs.currentPointer += int64(n)
		if err != nil && err != io.EOF {
			return processedSize, fmt.Errorf("error while reading data from stream: %w", err)
		}
		
		if processedSize == len(p) {
			return processedSize, nil
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

	var absPos int64
	switch whence {
	case seekStart:
		absPos = offset
	case seekCurrent:
		absPos = cs.currentPointer + offset 
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

	_, err :=  cs.streams[streamIndex].Seek(localOffset, seekStart)
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
	cs.indexCurrentStream = streamIndex

	return absPos, nil
}

func (cs *CombinedStream) TotalSize() int64 {
	return cs.totalSize
}

func (cs *CombinedStream) Close() error {
	var resultError error

	for _, stream := range cs.streams {
		errCur := stream.Close()
		resultError = errors.Join(resultError, errCur)
	}
	return resultError
}
