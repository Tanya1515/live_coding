package main

import (
	"fmt"
)

func Add(s []string) {
	s = append(s, "x") // записали "x"
}

func main() {
	s := []string{"a", "b", "c"}
	Add(s[1:2]) // передали "b"
	fmt.Println(s) // [a b x]
}
