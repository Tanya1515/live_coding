package main

import (
	"context"
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

type Response struct {
	StatusCode int
	Body       []byte
	Headers    map[string]string
}

type APIClient interface {
	Do(ctx context.Context, req Request) (*Response, error)
	Close() error
}

type APIClientEx struct {
	controloChan chan struct{}
	stopChan     chan struct{}
	config       APIClientConfig
	mu           *sync.RWMutex
	wg           *sync.WaitGroup
}

func NewApiClient(config APIClientConfig) *APIClientEx {
	controlChan := make(chan struct{}, config.RateLimit)
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
		mu:           &sync.RWMutex{},
		controloChan: controlChan,
		config:       config,
		stopChan:     stopChan,
		wg:           wg,
	}
}

func (api *APIClientEx) Do(ctx context.Context, req Request) (*Response, error) {
	// разделение на параметры request-а - 
	// а дальше отправка в соответсвующий канал 
	// для отправки запроса на сервер

	// два канала

	// process request
	return nil, nil
}

func (api *APIClientEx) DoReq(ctx context.Context, req Request) (*Response, error) {
	api.wg.Add(1)

	defer api.wg.Done()

	resp := &Response{}
	var err error
	api.mu.RLock()
	timeout := api.config.DefaultTimeout
	amountOfReauests := api.config.MaxRetries
	api.mu.RUnlock()

	ticker := time.NewTicker(timeout)
	defer ticker.Stop()

	select {
	case api.controloChan <- struct{}{}:
		resp, err = api.Do(ctx, req)
		if err == nil {
			return resp, err
		}
		if (resp.StatusCode == 429 || resp.StatusCode/100 == 5) && err != nil {
			amountOfReauests--
		} else {
			return resp, err
		}
	default:
	}

	for {
		select {
		case <-api.stopChan:
			return resp, err
		case <-ctx.Done():
			return resp, err
		case <-ticker.C:
			select {
			case api.controloChan <- struct{}{}:
				resp, err = api.Do(ctx, req)
				if amountOfReauests == 0 {
					return resp, err
				}
				if err == nil {
					return resp, err
				}
				if (resp.StatusCode == 429 || resp.StatusCode/100 == 5) && err != nil {
					timeout *= 2
					ticker.Reset(timeout)
					amountOfReauests--
				} else {
					return resp, err
				}
			default:
			}

		}
	}
}

func (api *APIClientEx) Close() error {
	close(api.stopChan)
	api.wg.Wait()
	close(api.controloChan)
	return nil
}
