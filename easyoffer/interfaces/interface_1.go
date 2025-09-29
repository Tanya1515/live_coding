package main

import (
	"errors"
	"fmt"
	"os"
)

func someFunc(shouldReturnErr bool) error {
	var err *os.PathError = nil

	if shouldReturnErr {
		return errors.New("time to throw error")
	}

	return err
}

func main() {
	errTrue := someFunc(true)
	fmt.Println(errTrue) // "time to throw error"
	fmt.Println(errTrue == nil) // false

	fmt.Println() // \n

	errFalse := someFunc(false)

	/* 

	1) fmt проверяет: errFalse != nil? → false (интерфейс не nil)

	2) fmt использует рефлексию: reflect.ValueOf(errFalse).IsNil() → true

	3) Видит что значение внутри интерфейса - nil указатель

	4) Печатает <nil> и НЕ вызывает Error()

	*/

	fmt.Println(errFalse) // nil
	fmt.Println(errFalse == nil) // false
}
