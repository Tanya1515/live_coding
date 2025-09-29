package main

import (
	"fmt"
	"runtime"
	"time"
)

func main() {
	for i := 0; i < 100; i++ {
		requestData(1)
	}

	time.Sleep(time.Second * 1)
	fmt.Printf("Number of hanging goroutines: %d\n", runtime.NumGoroutine()) // 101
}

func requestData(timeout time.Duration) string {
	dataChan := make(chan string)

	go func() {
		// горутина 1 раз вызывает функцию requestFromSlowServer и записывает строку в канал dataChan
		dataChan <- requestFromSlowServer()
	}()

	select {
	case result := <-dataChan:
		fmt.Printf("[+] request returned: %s\n", result) // very important data
		return result
	case <-time.After(timeout):
		fmt.Println("[!!] request timeout!")
		return ""
	}
}

func requestFromSlowServer() string {
	time.Sleep(time.Second * 1)
	return "very important data"
}

/*

Будет напечатано:

[!!] request timeout! 100 раз
Number of hanging goroutines: 101

Почему: 

В Go time.Duration - это алиас для int64, где 1 означает 1 наносекунду, а не 1 секунда!
Поэтому time.After будет отрабатывать каждый раз быстрее. 

Количество горутин будет равно 101, поскольку каждая горутина запишет в канал, но не завершит 
исполнение функции - канал небуферизованный, а значит горутина заблокируется и будет утечка. 
В итоге 100 зависших горутин + 1 main-горутина.

*/
