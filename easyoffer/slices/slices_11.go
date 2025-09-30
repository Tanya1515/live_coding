package main

import "fmt"

func main() {
	a := []int{1, 2}
	b := a[:]
	b = append(b, 3, 4, 5)
	b[0] = 2
	fmt.Println(a[0]) // 1
}
