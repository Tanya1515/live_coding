package main

import (
	"fmt"
)

type Example struct {
}
type MyInterface interface{}

func example() MyInterface { // пустой интерфейс
	var e *Example
	return e
}

func example2() MyInterface {
	return nil
}

func main() {
	// Обе функции выводят nil потому что fmt.Println смотрит только на значение внутри интерфейса.
	fmt.Println(example()) // nil 
	fmt.Println(example2()) // nil
	//fmt.Println(example() - example2()) // кажется оператор неопределен. код не скомпилируется

}
