package main

import "fmt"

func a() {
	x := []int{}
	x = append(x, 0)  // 0
	x = append(x, 1)  // 0 1
	x = append(x, 2)  // 0 1 2
	x = append(x, 3)  // 0 1 2 3
	y := append(x, 4) // 0 1 2 3 4 - на этом этапе x и y ссылаются на разные участки памяти.
	z := append(x, 5) // 0 1 2 3 5 - здесь тоже создается независимый срез
	fmt.Println(y, z) // [ 0 1 2 3 4 ] [ 0 1 2 3 5 ]
}

func main() {
	a()
}
