package main

import (
	"context"
	"errors"
	"sync"
	"time"
)

/*

Полное условие: Создайте клиент для REST API с:

1) Ограничением 100 запросов/минута (распределение равномерное)

2) Приоритетом для urgent-запросов - каналы + очередь с приоритетом

3) Автоматическим retry для 429/5xx ошибок

Требования:

1) Использовать token bucket для rate limiting

2) Retry с экспоненциальным backoff (max 3 попытки)

3) Приоритет urgent-запросов без starvation обычных

4) Метрики: кол-во запросов, ошибок, latency

Тесты:

 - Проверка лимитов

 - Очередь при перегрузке

 - Отмена через контекст

*/

type APIClientConfig struct {
	BaseURL        string
	DefaultTimeout time.Duration
	MaxRetries     int
	RateLimit      int // requests per minute - размер token bucket
}

type Request struct {
	Method   string
	Endpoint string
	Body     []byte
	Headers  map[string]string
	Urgent   bool
}

type ClientRequest struct {
	req        Request
	resultChan chan Result
	context    context.Context
}

type Response struct {
	StatusCode int
	Body       []byte
	Headers    map[string]string
}

type Result struct {
	err      error
	response *Response
}

type APIClient interface {
	Do(ctx context.Context, req Request) (*Response, error)
	Close() error
}

type APIClientEx struct {
	controlChan chan struct{}
	stopChan    chan struct{}
	urgentChan  chan ClientRequest
	generalChan chan ClientRequest
	config      APIClientConfig
	mu          *sync.RWMutex
	wg          *sync.WaitGroup
}

func NewApiClient(config APIClientConfig) *APIClientEx {
	controlChan := make(chan struct{}, config.RateLimit)
	urgentChan := make(chan ClientRequest, config.RateLimit)
	generalChan := make(chan ClientRequest, config.RateLimit)
	stopChan := make(chan struct{})

	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		defer wg.Done()
		for {
			select {
			case <-stopChan:
				return
			case <-ticker.C:
				for {
					select {
					case <-controlChan:
					default:
						break
					}
				}
			}
		}
	}()

	return &APIClientEx{
		mu:          &sync.RWMutex{},
		controlChan: controlChan,
		generalChan: generalChan,
		urgentChan:  urgentChan,
		config:      config,
		stopChan:    stopChan,
		wg:          wg,
	}
}

func (api *APIClientEx) Do(ctx context.Context, req Request) (*Response, error) {

	clientRequest := ClientRequest{req: req, context: ctx}
	clientRequest.resultChan = make(chan Result, 1)

	api.wg.Add(1)
	defer api.wg.Done()

	select {
	case <-api.stopChan:
		return nil, errors.New("API Client has been stopped")
	default:
		if req.Urgent {
			api.urgentChan <- clientRequest
		} else {
			api.generalChan <- clientRequest
		}
	}

	api.wg.Add(1)

	go api.doReq()

	for {
		select {
		case <-api.stopChan:
			return nil, errors.New("API Client has been stopped")
		case result := <-clientRequest.resultChan:
			return result.response, result.err
		}
	}
}

func (api *APIClientEx) doReq() {

	defer api.wg.Done()

	var req ClientRequest
	var err error
	var resp *Response

	api.mu.RLock()
	timeout := api.config.DefaultTimeout
	amountOfReauests := api.config.MaxRetries
	api.mu.RUnlock()

	ticker := time.NewTicker(timeout)
	defer ticker.Stop()

	select {
	case req = <-api.urgentChan:
	default:
		req = <-api.generalChan
	}
	defer close(req.resultChan)

	select {
	case api.controlChan <- struct{}{}:
		resp, err = api.processRequestOnServer(req.req)
		if err == nil {
			req.resultChan <- Result{response: resp, err: err}
			return
		}
		if (resp.StatusCode == 429 || resp.StatusCode/100 == 5) && err != nil {
			amountOfReauests--
		} else {
			req.resultChan <- Result{response: resp, err: err}
			return
		}
	default:
	}

	for {
		select {
		case <-api.stopChan:
			req.resultChan <- Result{response: resp, err: err}
			return
		case <-req.context.Done():
			req.resultChan <- Result{response: resp, err: err}
			return
		case <-ticker.C:
			select {
			case api.controlChan <- struct{}{}:
				resp, err = api.processRequestOnServer(req.req)
				if amountOfReauests == 0 {
					req.resultChan <- Result{response: resp, err: err}
					return
				}
				if err == nil {
					req.resultChan <- Result{response: resp, err: err}
					return
				}
				if (resp.StatusCode == 429 || resp.StatusCode/100 == 5) && err != nil {
					timeout *= 2
					ticker.Reset(timeout)
					amountOfReauests--
				} else {
					req.resultChan <- Result{response: resp, err: err}
					return
				}
			default:
			}
		}
	}
}

func (api *APIClientEx) processRequestOnServer(req Request) (*Response, error) {
	// process request
	return &Response{}, nil
}

func (api *APIClientEx) Close() error {
	close(api.stopChan)
	api.wg.Wait()
	close(api.controlChan)
	close(api.urgentChan)
	close(api.generalChan)
	return nil
}
