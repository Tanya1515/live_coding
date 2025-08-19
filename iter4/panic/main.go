package main

import "fmt"

func main() {
	defer fmt.Println("Defer 1")
	defer fmt.Println("Defer 2")

	func() {
		defer fmt.Println("Inner defer")

		panic("Panic!")
	}()

	fmt.Println("This won't print")
}


// Шаги исполнения программы: 
// 1) Будет отложен вызов fmt.Println("Defer 1") 
// 2) Будет отложен вызов fmt.Println("Defer 2") 
// 3) Будет отложен вызов fmt.Println("Inner defer") 
// 4) завершение программы 
// 5) вывод в консоль - "Inner defer" 
// 6) вывод в консоль - "Defer 2" 
// 7) вывод в консоль - "Defer 1" 
// 8) вывод в консоль информации о панике

