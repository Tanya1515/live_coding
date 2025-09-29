package main

import (
	"fmt"
  )
  
  // slices
  func main() {
	slice := make([]int64, 0, 4)
  
	slice = append(slice, 1) // 1 0 0 0; len = 1, cap = 4
	slice = append(slice, 2) // 1 2 0 0; len = 2, cap = 4
  
	fmt.Println(slice) // 1 2 
  
	slice = append2(slice, 3) // 1 2 3 0 
  
	fmt.Println(slice) // 1 2 3
  
	mutate2(slice, 2, 4) // panic
  
	fmt.Println(slice) // 1 2 4 
  }

  func append2(in []int64, value int64) []int64 {
	// здесь меняется длина в копии структуры слайса, которая 
	// уничтожается после завершения функции. 
	in = append(in, value) 
	return in
  }
  
  func mutate2(in []int64, idx, value int64) {
	in[idx] = value
  }