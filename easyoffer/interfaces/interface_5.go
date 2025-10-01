package main

import "fmt"

func main() {
	var s = "string"
	// не скомпилируется, поскольку строку нельзя сравнивать с nil
	//fmt.Println(s == nil)
	var i interface{}
	fmt.Println(i == nil) // true
	i = s
	fmt.Println(i == nil) // false
}
