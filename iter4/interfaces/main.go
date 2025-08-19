package main

import "fmt"

type MyError struct{}

func (e *MyError) Error() string {
	return "error"
}

func foo() error {
	var err *MyError
	return err
}

func main() {
	err := foo()
	fmt.Println(err == nil) // false, поскольку у err будет тип 
	
	// Когда fmt.Printf форматирует значение интерфейса error, он вызывает метод Error() 
	// для получения строкового представления. Поскольку err имеет тип *MyError, 
	// вызывается метод Error() структуры MyError. В Go вызов метода на nil-указателе допустим, 
	// если метод не использует поля структуры. В данном случае метод Error() для *MyError просто 
	// возвращает строку "error", не обращаясь к полям структуры. Поэтому fmt.Printf выводит error.
	fmt.Printf("%v\n", err) 
}