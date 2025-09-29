package main

import (
	"fmt"
)

type errorString struct {
	s string
}

func (e errorString) Error() string {
	return e.s
}

func checkErr(err error) {
	fmt.Println(err == nil)
}

// (nil, nil)        → true
// (*errorString, nil) → false
// (*errorString, &errorString{}) → false
// (*errorString, nil) → false

func main() {
	var e1 error
	checkErr(e1) // true

	var e *errorString
	checkErr(e) // false

	e = &errorString{}
	checkErr(e) // false

	e = nil     // значение: nil, но тип остаётся *errorString
	checkErr(e) // false
}
