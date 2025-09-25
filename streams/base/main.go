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
	readSize := len(p)
	if readSize == 0 {
		return 0, nil
	}
	if cs.TotalSize() == 0 {
		return 0, io.EOF
	}
	processedSize := 0

	for i := cs.indexCurrentStream; i < len(cs.streams); i++ {
		n, err := cs.streams[i].Read(p[processedSize : processedSize+readSize])
		if err != nil && err != io.EOF {
			processedSize += n
			cs.currentPointer += int64(n)
			return processedSize, fmt.Errorf("error while reading data from stream: %v", err)
		}
		processedSize += n
		cs.currentPointer += int64(n)
		readSize -= n
		if readSize == 0 {
			return processedSize, nil
		}

	}

	return processedSize, io.EOF
}

// выставить позицию окончания обработки потоков + поддержка отрицательных offset?
func (cs *CombinedStream) Seek(offset int64, whence int) (int64, error) {
	if cs.TotalSize() == 0 {
		return -1, fmt.Errorf("error: no data in stream has been found")
	}
	switch whence {
	case seekStart:
		for i := 0; i < len(cs.streams); i++ {
			streamSize := cs.streams[i].TotalSize()
			if streamSize > offset {
				cs.indexCurrentStream = i
				cs.currentPointer += offset
				cs.streams[i].Seek(offset, seekStart)
				break
			} else {
				cs.streams[i].Seek(streamSize, seekStart)
				offset -= streamSize
				cs.currentPointer += streamSize
			}
		}
	case seekCurrent:
		var size int64
		for i := 0; i < cs.indexCurrentStream; i++ {
			size += cs.streams[i].TotalSize()
		}
		currentStreamOffset := cs.currentPointer - size
		streamSize := cs.streams[cs.indexCurrentStream].TotalSize() - currentStreamOffset
		for i := cs.indexCurrentStream + 1; i < len(cs.streams); i++ {			
			if streamSize > offset {
				cs.indexCurrentStream = i
				cs.currentPointer = cs.currentPointer + offset
				cs.streams[i].Seek(offset, seekCurrent)
				break
			} else {
				offset -= streamSize
				cs.streams[i].Seek(streamSize, seekStart)
				cs.currentPointer += streamSize
			}
			streamSize = cs.streams[i].TotalSize()
		}
	case seekEnd:
		cs.currentPointer = cs.TotalSize()
		for i := len(cs.streams) - 1; i >= 0; i-- {
			streamSize := cs.streams[i].TotalSize()
			if streamSize > offset {
				cs.indexCurrentStream = i
				cs.currentPointer += streamSize + offset
				cs.streams[i].Seek(offset, seekEnd)
				break
			} else {
				offset += streamSize
				cs.streams[i].Seek(streamSize, seekStart)
				cs.currentPointer -= streamSize
			}
		}
	default:
		return 0, fmt.Errorf("invalid whence value")
	}
	if offset > 0 {
		return cs.currentPointer, io.EOF
	}
	return cs.currentPointer, nil
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
