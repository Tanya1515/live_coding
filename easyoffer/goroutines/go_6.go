package main

import (
	"fmt"
)

// Ничего не будет выводится, поскольку main-горутина раньше закончит работу. 

func main() {
	for i := 0; i < 5; i++ {
		go func() {
			fmt.Println(i)
		}()
	}
}
