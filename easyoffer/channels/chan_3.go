package main

import "fmt"

func main() {
	msgChan := make(chan string, 1)
	close(msgChan)
	stopChan := make(chan struct{})
	close(stopChan)

	select {
	case msgChan <- "msg": // Panic - нельзя писать в закрытый канал.
		fmt.Println("msg sent")
	case <-stopChan:
		fmt.Println("stop signal received")
	}
}
