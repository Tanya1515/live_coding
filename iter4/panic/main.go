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
