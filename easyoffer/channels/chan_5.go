package main

import (
	"fmt"
)

func Display(c chan string) {
	fmt.Printf("Display message: %s\n", <-c) // чтение из nil-канала → блокировка навсегда
}

func main() {
	fmt.Println("the programm has been started")

	var ch chan string // nil-канал
	go Display(ch)

	ch <- "first message" // panic - нельзя писать в неинициализированный канал
	ch <- "second message"
	fmt.Println("the programm has been has been finished")
}
