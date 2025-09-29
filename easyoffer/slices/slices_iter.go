package main

import (
	"fmt"
  )
  
  func main() {
	var numbers []*int
	for _, value := range []int{10, 20, 30, 40} {
	  numbers = append(numbers, &value)
	}
  
	// будет напечатано 10 20 30 40 для go 1.22+
	// для go 1.21 и ниже - 40 40 40 40
	for _, number := range numbers {
	  fmt.Printf("%d ", *number) 
	}
  }