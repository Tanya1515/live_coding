package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// Задача:
// 1) Произвести обработку ссылок, отправив GET запросы на URL и выводить в консоль 200 статус ответа или иной
// 2) Распараллелить обработку ссылок
// 3) Реализовать остановку обработки после получения извне сигнала о прекращении работы

func Worker(wg *sync.WaitGroup, ctx context.Context, chanUrl chan string) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case url, ok := <-chanUrl:
			if !ok {
				return
			}
			resp, err := http.Get(url)
			if err != nil {
				fmt.Println(err.Error())
			} else {
				fmt.Println(url, resp.StatusCode)
				resp.Body.Close()
			}
		}
	}
}

func main() {

	var urls = []string{
		"http://ozon.ru",
		"https://ozon.ru",
		"http://google.com",
		"http://somesite.com",
		"http://non-existent.domain.tld",
		"https://ya.ru",
		"http://ya.ru",
		"http://ёёёё",
	}

	ctx, cancel := context.WithCancel(context.Background())

	chanUrl := make(chan string, 5)

	gracefullShutdown := make(chan os.Signal, 1)
	signal.Notify(gracefullShutdown, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-gracefullShutdown
		cancel()
	}()

	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go Worker(&wg, ctx, chanUrl)
	}

	go func() {
		for _, url := range urls {
			chanUrl <- url
		}

		close(chanUrl)
	}()

	wg.Wait()

}
