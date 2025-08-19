package main

import "fmt"

type MyInt int

func (i MyInt) Val() int {
	return int(i)
}

func (i *MyInt) Inc() {
	*i++
}

func main() {
	var x MyInt = 10

	fmt.Println(x.Val())

	x.Inc()

	fmt.Println(x.Val())
}
