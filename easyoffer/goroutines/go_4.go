package main

import (
	"fmt"
)

// Здесь будет data race, поскольку несинхронизированно 
// обращение к разделяемой переменной. 

func main() {
	var max int

	for i := 1000; i > 0; i-- {
		go func() {
			if i%2 == 0 && i > max {
				max = i
			}
		}()
	}
	fmt.Printf("Maximum is %d", max) // напечатается 1000, но поведение может быть и другим из-за гонки данных
}
