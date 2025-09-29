package main

import (
	"fmt"
	"time"
)

/*

Функция time.After(d) в Go возвращает канал, 
в который по истечении времени d (указанной продолжительности) 
будет отправлено текущее время. 

*/

func main() {
	ch := make(chan int)
	go func() {
		for i := 0; i < 5; i++ {
			ch <- i
			time.Sleep(time.Second)
		}
		close(ch)
	}()

	for {
		select {
		case v := <-ch:
			fmt.Println(v) 
		case <-time.After(3 * time.Second):
			fmt.Println("timeout") 
			break // выполнится из select-а
		}
	}
}

// 0 1 2 timeout 3 4 timeout 0 0 0 0 0 0.. - как дефолтное значение из канала. 