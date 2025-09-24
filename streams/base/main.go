package main

import "io"

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
 */

/*
type io.ReadSeekCloser interface {
    Read(p []byte) (n int, err error)
    Seek(offset int64, whence int) (int64, error)
    Close() error
}
*/

type MeasuredStream interface {
	io.ReadSeekCloser
	TotalSize() int64
}

type CombinedStream struct {
	// put your code here...
}

func NewCombinedStream(streams ...MeasuredStream) *CombinedStream {
	// put your code here...
	return nil
}
