package main

import (
	"fmt"
)

type MyErr struct{}

func (me MyErr) Error() string {
	return "my err string"
}

// Интерфейс считается nil только если и тип и значение равны nil!

func main() {
	fmt.Println(returnError() == nil)    // true
	fmt.Println(returnErrorPtr() == nil) // true

	fmt.Println(returnCustomError() == nil)    // false
	fmt.Println(returnCustomErrorPtr() == nil) // false

	fmt.Println(returnMyError() == nil) // true
}


/*

err объявлен как error (интерфейс)

Нулевое значение интерфейса - это (nil, nil)

Результат: чистый nil интерфейс

*/
func returnError() error {
	var err error
	return err
}

// здесь возвращается указатель на nil
func returnErrorPtr() *error {
	var err *error
	return err
}

func returnCustomError() error {
	var customErr MyErr
	return customErr
}

func returnCustomErrorPtr() error {
	var customErr *MyErr
	return customErr
}

func returnMyError() *MyErr {
	return nil
}
