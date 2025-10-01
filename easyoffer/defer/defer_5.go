package main

import (
	"fmt"
)

func main() {
	i1 := 10
	k := 20
	i2 := &k

	defer printInt("i1", i1)                 // 3) i1 = 10
	defer printIntPointer("i2 as value", i2) // 2) i2 as value 2020
	defer printInt("i2 as pointer", *i2)     // 1) i2 as pointer: 20

	i1 = 1010
	*i2 = 2020

	fmt.Println(k) // 2020
}

func printInt(v string, i int) {
	fmt.Printf("%s=%d\n", v, i)
}

func printIntPointer(v string, i *int) {
	fmt.Printf("%s=%d\n", v, *i)
}
