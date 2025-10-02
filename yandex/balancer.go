package main

import (
	"context"
	"time"
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
2) Исключать сбоящий экземпляр из балансировки
3) Распределять нагрузку равномерно с учетом загруженности

Основные шаги решения:

1) Сделать отдельную горутину, которая будет проверять состояние каждого из бэкендов в отдельной горутине.
То есть заводим фиксированное количество горутин равное количеству backend-ов и рассылаем запросы. Каждая
из горутин будет модифицировать поле state у backend-а.

2) В методе Invoke будем брать backend, который лежит по индексу next. Если его состояние закрытое (closed),
тогда ему можно отправлять запросы. Также проверяем amountOfRequests и записываем в next балансировщик,
у которого минимальное amountOfRequests, а также состояние closed или half open.

3) Для каждого backend-а будем хранить количество запросов, которые отработали с ошибками. Если количество запросов с ошибками
превышает определенный порог - то обнуляем количество ошибок, выставялем состояние в open и запускаем ticker, по которому
будем отправлять healthcheck.

*/

type Request interface{}
type Response interface{}
type healthCheckRequest struct{}
type Backend interface {
	Invoke(ctx context.Context, req Request) (Response, error)
}
type State int
const (
	open State = iota
	closed
)
const (
	criticalAmountErr = 100
	workers           = 10
)
type BackendImpl struct {
	addr            string
	state           State
	amountOfProcReq int
	amountOfErr     int
}
func NewBackend(addr string) *BackendImpl {
	return &BackendImpl{
		addr:            addr,
		state:           closed,
		amountOfErr:     0,
		amountOfProcReq: 0,
	}
}

func (b BackendImpl) Invoke(ctx context.Context, req Request) (Response, error) {
	return nil, nil
}

func (b BackendImpl) healthCheck(ctx context.Context, req healthCheckRequest) error {
	return nil
}

var _ Backend = &BackendImpl{}

type Balancer struct {
	addrs    []string
	backends []BackendImpl
	mu       *sync.RWMutex
	wg       *sync.WaitGroup
	next     int
	stopChan chan struct{}
}

func (b *Balancer) processBackend(backChan chan int) {
	defer b.wg.Done()

	var req healthCheckRequest
	var err error
	for backInd := range backChan {
		b.mu.Lock()
		back := b.backends[backInd]
		b.mu.Unlock()

		for i := 0; i < 3; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			err := back.healthCheck(ctx, req)
			if err == nil {
				break
			}
			time.Sleep(time.Duration(i) * time.Second)
		}

		if err == nil {
			b.mu.Lock()
			b.backends[backInd].state = closed
			b.mu.Unlock()
		}

	}

}

func (b *Balancer) checkBackends() {
	defer b.wg.Done()

	inputChan := make(chan int, 10)

	for i := 0; i < workers; i++ {
		b.wg.Add(1)
		go b.processBackend(inputChan)
	}

	for {
		select {
		case <-b.stopChan:
			close(inputChan)
			return
		default:
			mapBack := make(map[BackendImpl]int)
			b.mu.RLock()
			for index, back := range b.backends {
				if back.state == closed && back.amountOfErr < criticalAmountErr {
					continue
				}
				mapBack[back] = index
			}
			b.mu.RUnlock()

			for back, ind := range mapBack {
				if back.amountOfErr >= criticalAmountErr {
					b.mu.Lock()
					b.backends[ind].state = open
					b.backends[ind].amountOfErr = 0
					b.mu.Unlock()
				}
				inputChan <- ind
			}
		}
	}

}

func NewBalancer(addrs []string) *Balancer {
	backends := make([]BackendImpl, 0)
	stopChan := make(chan struct{})
	for _, addr := range addrs {
		back := NewBackend(addr)
		backends = append(backends, *back)
	}

	balancer := &Balancer{
		addrs:    addrs,
		backends: backends,
		next:     0,
		mu:       &sync.RWMutex{},
		wg:       &sync.WaitGroup{},
		stopChan: stopChan,
	}

	balancer.wg.Add(1)
	go balancer.checkBackends()

	return balancer
}

func (b *Balancer) nextBackend() {
	backInd := 0
	minReq := -1

	defer b.wg.Done()

	b.mu.RLock()
	for index, back := range b.backends {

		select {
		case <-b.stopChan:
			b.mu.RUnlock()
			return
		default:
			if back.state == open {
				continue
			}
			if minReq == -1 || minReq > back.amountOfProcReq {
				minReq = back.amountOfProcReq
				backInd = index
			}
		}
	}

	b.mu.RUnlock()

	b.mu.Lock()
	b.next = backInd
	b.mu.Unlock()
}

func (b *Balancer) Invoke(ctx context.Context, req Request) (Response, error) {

	b.mu.Lock()
	back := &b.backends[b.next]
	b.backends[b.next].amountOfProcReq++
	b.mu.Unlock()

	b.nextBackend()

	resp, err := back.Invoke(ctx, req)

	b.mu.Lock()
	back.amountOfProcReq--
	if err != nil {
		back.amountOfErr++
	}
	b.mu.Unlock()

	return resp, err
}

func (b *Balancer) Stop() {
	close(b.stopChan)
	b.wg.Wait()
}
