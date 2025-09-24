package main

import "io"

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

type MeasuredStream interface {
	io.ReadSeekCloser
	TotalSize() int64
}

type CombinedStream struct {
	// put your code here...
}

func NewCombinedStream(buffersNum int, rs ... MeasuredStream) * CombinedStream {
    // put your code here...
    return nil
}