package main

import (
	"sync/atomic"
	"time"
)

// Задача:

// Создайте структуру RateLimiter, которая ограничивает количество вызовов функции
// в указанный промежуток времени (например, не более 5 вызовов в секунду).

type RateLimiter struct {
	Rate              int
	Interval          time.Duration
	FunctionCallCount int32
	StopChannel       chan struct{}
}

func NewRateLimiter(rate int, interval time.Duration) *RateLimiter {
	// валидация значений (возможно, не нужна)
	// вернуть RateLimiter с пустыми значениями
	if rate <= 0 {
		return &RateLimiter{}
	}
	if interval <= 0 {
		return &RateLimiter{}
	}
	stopChannel := make(chan struct{})
	rateLimiter := &RateLimiter{
		Rate:              rate,
		Interval:          interval,
		FunctionCallCount: 0,
		StopChannel:       stopChannel,
	}
	go func(rateLimiter *RateLimiter) {
		ticker := time.NewTicker(rateLimiter.Interval)
		for {
			select {
			case _, closed := <-rateLimiter.StopChannel:
				if closed {
					ticker.Stop()
					return
				}
			case <-ticker.C:
				atomic.StoreInt32(&rateLimiter.FunctionCallCount, 0)
			}
		}

	}(rateLimiter)
	return rateLimiter
}

func (r *RateLimiter) Call(fn func()) bool {
	if atomic.CompareAndSwapInt32(&r.FunctionCallCount, int32(r.Rate), int32(r.Rate)) {
		return false
	} else {
		atomic.AddInt32(&r.FunctionCallCount, 1)
		go fn()
	}
	return true
}

// Stop используется для очистки ресурсов, используемых в Rate Limiter
func (r *RateLimiter) Stop() {

	close(r.StopChannel)
}
