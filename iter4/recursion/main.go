package main

import (
	"fmt"
	"sync"
)

type Counter struct {
	mu  sync.Mutex
	val int
}

func (c *Counter) Inc() {
	c.mu.Lock()

	defer c.mu.Unlock()

	c.val++

	if c.val < 10 {
		// c.mu.Unlock() - убрать defer, добавить сюда снятие мьютекса
		c.Inc()
	}
}

func main() {
	c := &Counter{}

	c.Inc() // функция Inc вызывается рекурсивно, однако, мьютекс разлочится только при выходе из функции. При этом сама функция рекурсивна.

	fmt.Println(c.val)
}
