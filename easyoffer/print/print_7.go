package main

import (
	"fmt"
)

type myError struct {
	code int
}

// failed to run, error: nil

func (e myError) Error() string {
	return fmt.Sprintf("code: %d", e.code)
}

func run() error {
	var e *myError
	if false { // никогда не выполнится 
		e = &myError{code: 123}
	}
	return e
}

func main() {
	// err не равен nil, поскольку в интерфейсе указан тип
	err := run()
	if err != nil {
		fmt.Println("failed to run, error:", err)
	} else {
		fmt.Println("success")
	}

}
