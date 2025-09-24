package main

// Пошагово sync.Once работает следующим образом: 
// 1) Первая горутина доходит до c.once.Do(): 
// 		- захватывает внутренний мьютекс 
//		- выполняет функцию инициализации 
// 		- устанавливает внутренний флаг "выполнено" 
// 		- освобождает мьютекс 
// 2) Последующие горутины доходят до c.once.Do() 
// 		- проверяют флаг выполнено 
// 		- немедленно продолжают выполнение без блокировки 

import (
    "fmt"
    "math/rand"
    "sync"
    "time"
)

type Config struct {
    once sync.Once
    vals map[string]string
}

func (c *Config) Get(k string) (string, bool) {
    c.once.Do(func() {
        // эта инициализация выполнится только один раз
        c.vals = map[string]string{
            "host": "127.0.0.1",
            "port": fmt.Sprintf("%d", rand.Intn(65535)),
        }
    })

    v, ok := c.vals[k]
    return v, ok
}

func main() {
    var cfg Config

    keys := []string{"host", "port", "port", "host", "port"}
    for _, k := range keys {
        go func(k string) {
            // в одной из горутин произойдёт инициализация cfg
            // остальные горутины будут ждать завершения инициализации
            v, ok := cfg.Get(k)
            if !ok {
                return
            }
            fmt.Printf("%s = %s\n", k, v)
        }(k)
    }

    time.Sleep(1 * time.Second)
} 