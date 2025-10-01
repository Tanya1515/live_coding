package main

import "fmt"

func main() {
	var strData string
	for i := 0; i < 10; i++ {
		strData += fmt.Sprintf("%d", i)
	}
	fmt.Println(strData) // 0123456789
}
