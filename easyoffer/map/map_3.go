package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Race condition не будет, н лучше синхронизировать завершение горутин,
// например, при помощи контекста.

var (
	mu sync.RWMutex
	m  = map[string]int{"a": 1}
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	go read(ctx)
	time.Sleep(1 * time.Second)
	go write(ctx)
	time.Sleep(1 * time.Second)
	cancel()
}

func read(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			mu.RLock()
			fmt.Println(m["a"])
			mu.RUnlock()
		}
	}
}

func write(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			mu.Lock()
			m["a"] = 2
			mu.Unlock()
		}
	}
}
