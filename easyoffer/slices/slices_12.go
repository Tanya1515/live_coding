package main

import "fmt"

func main() {
	a := []int{1, 2, 3, 4, 5}

	for _, i := range a {
		go func() {
			// не будет ничего выведено, 
			// поскольку main-горутина раньше закончит выполнение
			fmt.Print(i) 
		}()
	}
}
