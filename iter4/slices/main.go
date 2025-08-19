package main

import "fmt"

func main() {
	s := make([]int, 0, 100)
	for i := 0; i < 10; i++ {
		s = append(s, i)
	}
	fmt.Println(len(s), cap(s)) // 10 100
	t := s[:5]
	t = append(t, 99)
	fmt.Println(s)

}
