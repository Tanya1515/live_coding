package main

import (
	"fmt"
)

func main() {
	var m map[string]int

	fmt.Println(m["foo"]) // 0

	m["foo"] = 42 // panic-а попытка записи в неинициализированную мапу

	fmt.Println(m["foo"])
}
