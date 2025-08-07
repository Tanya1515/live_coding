package main

import (
	"context"
	"errors"
	"sync"
	"time"
)

/*
Есть приложение с микросервисной архитектурой.
Микросервис можно абстрагировать с помощью интерфейса Backend.
Для доступа к одному экземпляру микросервиса можно использовать
тип BackendImpl, который уже реализован.

Для каждого микросервиса есть несколько десятков запущенных
экземпляров, каждый из которых доступен по своему адресу addr.
Однако отдельные экземпляры микросервиса ненадежны:
они могут падать, быть недоступными либо перегруженными.
Поэтому вам нужно реализовать тип Balancer, который также реализует
интерфейс Backend и осуществляет client-side балансировку нагрузки
между экземплярами микросервиса.

1) Отправлять запрос в backend
2) Исключать сбоящий экземпляр из баласировки
3) Распределять нагрузку равномерно с учетом загруженности

Поставить таймер, сколько уйдет на каждую часть
Продумать часть с контекстом, с которым пришел пользователь

*/

type Request interface{}

type Response interface{}

type healthCheckRequest struct{}

type Backend interface {
	Invoke(ctx context.Context, req Request) (Response, error)
}

type BackendImpl struct {
	addr        string
	active      bool
	amountOfReq int
	mu          *sync.RWMutex
}

var _ Backend = &BackendImpl{}

func NewBackend(addr string) *BackendImpl {
	return &BackendImpl{addr: addr, active: true, amountOfReq: 0, mu: &sync.RWMutex{}}
}

func (back *BackendImpl) Invoke(ctx context.Context, req Request) (Response, error) {
	return nil, nil
}

func (back *BackendImpl) HealthCheck(ctx context.Context, req Request, stopChannel chan struct{}, syncChannel chan *BackendImpl, wg *sync.WaitGroup) {
	back.mu.Lock()
	back.active = false
	back.amountOfReq = 0
	back.mu.Unlock()
	defer wg.Done()

	ticker := time.NewTicker(1000 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_, err := back.Invoke(ctx, req)
			if err == nil {
				back.mu.Lock()
				back.active = true
				back.amountOfReq = 0
				back.mu.Unlock()
				syncChannel <- back
				return
			}
		case <-stopChannel:
			return
		case <-ctx.Done():
			return
		default:
		}
	}
}

type Balancer struct {
	backends    map[*BackendImpl]struct{}
	mu          *sync.RWMutex
	next        *BackendImpl
	syncChannel chan *BackendImpl
	stopChannel chan struct{}
	wg          *sync.WaitGroup
	minReq      int
}

// Balancer удовлетворяет интерфейсу Backend-а
var _ Backend = &Balancer{}

// addrs содержат адреса всех балансируемых экземпляров
func NewBalancer(addrs []string) *Balancer {
	backends := make(map[*BackendImpl]struct{}, 0)
	syncChannel := make(chan *BackendImpl, len(addrs))
	stopChannel := make(chan struct{})
	minReq := 0
	wg := &sync.WaitGroup{}
	mu := &sync.RWMutex{}

	firstBackend := NewBackend(addrs[0])
	for _, addr := range addrs {
		backend := NewBackend(addr)
		backends[backend] = struct{}{}
	}

	balancer := &Balancer{
		backends:    backends,
		mu:          mu,
		next:        firstBackend,
		syncChannel: syncChannel,
		stopChannel: stopChannel,
		wg:          wg,
		minReq:      minReq,
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case backend, ok := <-syncChannel:
				if !ok {
					return
				}
				mu.Lock()
				backends[backend] = struct{}{}
				mu.Unlock()
			case <-ticker.C:
				balancer.next.mu.RLock()
				tekBackend := balancer.next
				balancer.next.mu.RUnlock()

				if !tekBackend.active || (balancer.minReq != tekBackend.amountOfReq) {
					balancer.mu.RLock()
					minReqCount := -1

					for back := range balancer.backends {
						back.mu.RLock()
						if minReqCount == -1 && back.active {
							tekBackend = back
							minReqCount = back.amountOfReq
						}
						if minReqCount > back.amountOfReq && back.active {
							tekBackend = back
							minReqCount = back.amountOfReq
						}
						back.mu.RUnlock()
					}
					balancer.mu.RUnlock()

					balancer.mu.Lock()
					if _, exists := balancer.backends[tekBackend]; exists {
						balancer.next = tekBackend
						balancer.minReq = minReqCount
					}
					balancer.mu.Unlock()
				}
			default:
			}
		}
	}()

	return balancer
}

func (b *Balancer) Invoke(ctx context.Context, req Request) (Response, error) {

	if len(b.backends) == 0 {
		return nil, errors.New("No available backends found")
	}

	b.mu.RLock()
	backend := b.next
	b.mu.RUnlock()

	backend.mu.Lock()
	backend.amountOfReq += 1
	backend.mu.Unlock()

	resp, err := backend.Invoke(ctx, req)
	seconds := time.Second
	counter := 1
	for {
		if err == nil {
			backend.mu.Lock()
			if backend.active {
				backend.amountOfReq -= 1
			}
			backend.mu.Unlock()
			break
		}
		if counter == 3 {
			break
		}
		select {
		case <-ctx.Done():
			if err != nil {
				checkReq := healthCheckRequest{}
				b.wg.Add(1)
				b.mu.Lock()
				delete(b.backends, backend)
				b.mu.Unlock()
				go backend.HealthCheck(ctx, checkReq, b.stopChannel, b.syncChannel, b.wg)
			}
			return resp, err
		default:
			time.Sleep(seconds)
			seconds++
			resp, err = backend.Invoke(ctx, req)
			counter++
		}
	}

	if err != nil {
		checkReq := healthCheckRequest{}
		b.wg.Add(1)
		b.mu.Lock()
		delete(b.backends, backend)
		b.mu.Unlock()
		go backend.HealthCheck(ctx, checkReq, b.stopChannel, b.syncChannel, b.wg)
	}
	return resp, err
}

func (b *Balancer) Stop() {
	close(b.stopChannel)
	b.wg.Wait()
	close(b.syncChannel)
}
