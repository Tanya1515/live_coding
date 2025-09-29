package main

import "fmt"

/*

a 1
b 2
c 3

*/

func main() {
	m := map[string]int{"a": 1, "b": 2, "c": 3}

	for a, b := range m {
		fmt.Println(a, b)
	}

}
