package main

import "fmt"

func addElem(s []string) {
	s = append(s, "x") // b x
}

func main() {
	s := []string{"a", "b", "c"} // a b c, len = 3, cap = 3
	addElem(s[1:2]) // b, len = 1
	fmt.Println(s) // a b x
}
