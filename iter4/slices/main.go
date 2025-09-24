package main

import "fmt"

func main() {
	s := make([]int, 0, 100)
	for i := 0; i < 10; i++ {
		s = append(s, i)
	}
	fmt.Println(len(s), cap(s)) // 10 100
	t := s[:5]
	t = append(t, 99) // len(t) = 6, cap(t) = 100, len(s) = 10, cap(s) = 100
	fmt.Println(s) // 0 1 2 3 4 99 6 7 8 9 

}
