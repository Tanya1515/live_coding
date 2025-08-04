package main

import (
	"context"
	"sync"
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
2) Исключать сбоящий экземпляр из баласировки -> retry-механзим + таймер, сколько не трогаем backend
3) Распределять нагрузку равномерно с учетом загруженности

Поставить таймер, сколько уйдет на каждую часть 
Продумать часть с контекстом, с которым пришел пользователь

*/

type Request interface{}

type Response interface{}

type Backend interface {
	Invoke(ctx context.Context, req Request) (Response, error)
}

type BackendImpl struct {
	Addr string
}

var _ Backend = &BackendImpl{}

// addr содержит ip:port конкретного экземпляра
func NewBackend(addr string) *BackendImpl {
	return &BackendImpl{Addr: addr}
}

func (back *BackendImpl) Invoke(ctx context.Context, req Request) (Response, error) {
	return nil, nil
}

type Balancer struct {
	backends      []*BackendImpl
	mu *sync.Mutex
	next          int
}

// Balancer удовлетворяет интерфейсу Backend-а
var _ Backend = &Balancer{}

// addrs содержат адреса всех балансируемых экземпляров
func NewBalancer(addrs []string) *Balancer {
	backends := make([]*BackendImpl, 0)
	
	for _, addr := range addrs {
		backend := NewBackend(addr)
		backends = append(backends, backend)
	}
	return &Balancer{
		backends: backends,
		mu: &sync.Mutex{},
		next:     0,
	}
}

func (b *Balancer) Invoke(ctx context.Context, req Request) (Response, error) {

	backend := b.backends[b.next%len(b.backends)]

	b.mu.Lock()
	b.next = (b.next + 1)
	b.mu.Unlock()

	resp, err := backend.Invoke(ctx, req)

	return resp, err
}
