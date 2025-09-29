package main

import "fmt"

func main() {
	x := []int{}
	x = append(x, 0) // 0
	x = append(x, 1) // 0 1
	x = append(x, 2) // 0 1 2 
	y := append(x, 3) // 0 1 2 3
	z := append(x, 4) // 0 1 2 4
	fmt.Println(y, z) // [0 1 2 4] [0 1 2 4]
}
