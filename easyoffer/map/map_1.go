package main

import "fmt"

func main() {
	a := map[string]int{}
	b :=make(map[string]int, 10)

	a["test"]++
	b["test"]++

	fmt.Println(a["test"]) // 1
	fmt.Println(b["test"]) // 1
}
