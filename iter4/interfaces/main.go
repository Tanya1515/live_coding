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
	fmt.Println(err == nil)
	fmt.Printf("%v\n", err)
}
