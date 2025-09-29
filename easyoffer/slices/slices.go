package main

import (
	"fmt"
  )
  
  // slices
  func main() {
	slice := make([]int64, 0, 4)
  
	slice = append(slice, 1) // 1; len = 1, cap = 4
	slice = append(slice, 2) // 1 2 ; len = 2, cap = 4
  
	fmt.Println(slice) // 1 2 
  
	append1(slice, 3) // 1 2 
  
	fmt.Println(slice) // 1 2 
	
	// panic - попытка обратится к неинициализированной ячейке массива, 
	// поскольку пока в массиве присустсвуют элементы по индексам 0 и 1. 
	mutate1(slice, 3, 4) 
  
	fmt.Println(slice) 
  }
  
/*

Что происходит в append1:

1) in получает копию структуры слайса: length=2, capacity=4, pointer=0x1234

2) append(in, value) работает с этой копией

3) После append: in имеет length=3, capacity=4, pointer=0x1234

4) Но! Когда функция завершается, копия in уничтожается, 
а исходный слайс в main остается неизменным (length=2)

*/

  func append1(in []int64, value int64) {
	// здесь меняется длина в копии структуры слайса, которая 
	// уничтожается после завершения функции. 
	in = append(in, value) 
  }
  
  func mutate1(in []int64, idx, value int64) {
	in[idx] = value
  }