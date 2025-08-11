// You can edit this code!
// Click here and start typing.
package main

import "fmt"

// Что выведет код? 
// Правильный ответ: [0, 1, 2, 4] [0, 1, 2, 4]

func main() {
	a()
}

func a() {
	x := []int{}
	x = append(x, 0)
	x = append(x, 1)
	x = append(x, 2) 
	y := append(x, 3) 
	z := append(x, 4) 
	fmt.Println(y, z) 
}
