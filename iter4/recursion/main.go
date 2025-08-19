package main

import (
	"fmt"
	"sync"
)

type Counter struct {
	mu sync.Mutex
	val int
}

func (c *Counter) Inc() {
	c.mu.Lock()

	defer c.mu.Unlock()

	c.val++

	if c.val < 10 {
		c.Inc()
	}
}

func main() {
	c := &Counter{}

	c.Inc()

	fmt.Println(c.val)
}
