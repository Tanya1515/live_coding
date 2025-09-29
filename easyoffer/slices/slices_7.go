package main

import (
	"fmt"
)

func mod(a []int) {
	a = append(a, 125)

	for i := range a {
		a[i] = 5
	}

	fmt.Println(a) // 5 5 5 5
}

func main() {
	sl := []int{1, 2, 3, 4}
	mod(sl)
	fmt.Println(sl) // 5 5 5 5
}

/*

Версия с append:

5 5 5 5 5

1 2 3 4

*/
