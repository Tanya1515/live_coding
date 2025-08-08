package main

import (
	"context"
	"time"
)

/*

Полное условие: Создайте клиент для REST API с:

1) Ограничением 100 запросов/минута (распределение равномерное)

2) Приоритетом для urgent-запросов

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
	RateLimit      int // requests per minute
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
