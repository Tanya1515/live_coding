package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

/*

Полное условие:

Реализуйте HTTP-сервер с динамической конфигурацией, который:

1) Поддерживает graceful shutdown (завершает текущие соединения перед выходом)

2) Перезагружает конфиг без downtime при получении SIGHUP

3) Валидирует конфиг перед применением

Требования:

- Конфиг должен читаться из YAML-файла

- При невалидном конфиге - продолжать работать со старыми настройками

- Все текущие запросы должны завершиться или получить уведомление через контекст

Тесты должны проверять:

- Завершение long-polling запросов при shutdown

- Конкурентный доступ к конфигу

- Откат при ошибке валидации

// Пример обработчика с поддержкой контекста
type HandlerWithContext func(ctx context.Context, w http.ResponseWriter, r *http.Request)

*/

type Config struct {
	Addr         string        `yaml:"addr"`         // адрес, который слушает http-сервер
	ReadTimeout  time.Duration `yaml:"read_timeout"` // не очень понятно, что за таймауты ?
	WriteTimeout time.Duration `yaml:"write_timeout"`
	Workers      int           `yaml:"workers"` // количество worker-ов, которые обрабатывают запросы
}

type ConfigValidator interface {
	Validate() error
}

func (c *Config) Validate() error {
	if c.Addr == "" {
		return errors.New("Error: address is empty")
	}

	if c.ReadTimeout < 0 {
		return errors.New("Incorrect timeout for read operations")
	}

	if c.WriteTimeout < 0 {
		return errors.New("Incorrect timeout for write operations")
	}

	if c.Workers <= 0 {
		return errors.New("Incorrect amount of Workers")
	}
	return nil
}

func NewConfig(configPath string) (*Config, error) {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}

	err = yaml.Unmarshal(content, cfg)
	if err != nil {
		return nil, err
	}

	err = cfg.Validate()

	return cfg, err
}

type HandlerWithContext func(ctx context.Context, w http.ResponseWriter, r *http.Request)

type Server interface {
	Start() error
	Stop(ctx context.Context) error
	ReloadConfig(cfg Config) error
}

type ServerEx struct {
	cfg                *Config
	server             *http.Server
	serverHandlers     *http.ServeMux
	serverContext      context.Context
	serverCancel       context.CancelFunc
	workerPool         chan Task
	registeredHandlers map[string]HandlerWithContext
	mu                 *sync.RWMutex
	wg                 *sync.WaitGroup
}

func NewServer(cfg *Config, h map[string]HandlerWithContext) (*ServerEx, error) {

	serverCtx, serverCancel := context.WithCancel(context.Background())

	workerPool := make(chan Task, cfg.Workers)

	mux := http.NewServeMux()

	serv := &ServerEx{
		cfg:                cfg,
		serverContext:      serverCtx,
		serverCancel:       serverCancel,
		registeredHandlers: h,
		workerPool:         workerPool,
		mu:                 &sync.RWMutex{},
		wg:                 &sync.WaitGroup{},
		serverHandlers:     mux,
	}

	return serv, nil
}

func (s *ServerEx) serverWorker() {
	defer s.wg.Done()
	for {
		select {
		case <-s.serverContext.Done():
			return
		case task := <-s.workerPool:
			task.handler(task.ctx, task.wr, task.r)
		default:
		}
	}
}

func (s *ServerEx) Start() error {

	for i := 0; i < s.cfg.Workers; i++ {
		s.wg.Add(1)
		go s.serverWorker()
	}

	server := &http.Server{
		Addr:         s.cfg.Addr,
		Handler:      s.serverHandlers,
		ReadTimeout:  s.cfg.ReadTimeout,
		WriteTimeout: s.cfg.WriteTimeout,
	}

	s.server = server

	go func() {
		err := server.ListenAndServe()
		if err != nil {
			return
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGHUP, syscall.SIGTERM)

	go func() {
		for {
			select {
			case signal := <-sigChan:
				switch signal {
				case syscall.SIGINT:
					go func() {
						s.Stop(context.Background())
					}()
				case syscall.SIGTERM:
					go func() {
						s.Stop(context.Background())
					}()
				case syscall.SIGHUP:
					// непонятно, откуда получать конфиг
					s.ReloadConfig(Config{Addr: "localhost:8080", ReadTimeout: 1 * time.Second, WriteTimeout: 2 * time.Second, Workers: 3})
				}
			case <-s.serverContext.Done():
				return
			default:
			}
		}
	}()

	return nil
}

type Task struct {
	ctx     context.Context
	wr      http.ResponseWriter
	r       *http.Request
	handler HandlerWithContext
}

func (s *ServerEx) RegisterHandler(path string, handler HandlerWithContext) error {
	s.mu.RLock()

	if _, exists := s.registeredHandlers[path]; exists {
		s.mu.RUnlock()
		return errors.New("Error: handler with path " + path + " already exists")
	}
	s.mu.RUnlock()

	s.mu.Lock()
	s.registeredHandlers[path] = handler

	s.serverHandlers.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		for {
			select {
			case s.workerPool <- Task{r.Context(), w, r, handler}:
			case <-s.serverContext.Done():
				return
			}
		}
	})
	s.mu.Unlock()

	return nil
}

func (s *ServerEx) ReloadConfig(cfg Config) error {

	err := cfg.Validate()
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:         cfg.Addr,
		Handler:      s.serverHandlers,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	errChan := make(chan error, 1)
	go func() {
		errChan <- server.ListenAndServe()
		close(errChan)
	}()

	select {
	case <-errChan:
		return err
	case <-time.After(1 * time.Second):
	}

	s.mu.Lock()
	s.server.Shutdown(context.Background())
	s.cfg = &cfg
	s.server = server
	s.mu.Unlock()

	return nil
}

func (s *ServerEx) Stop(ctx context.Context) error {
	s.serverCancel()

	s.server.Shutdown(ctx)

	s.wg.Wait()

	close(s.workerPool)

	return nil
}
