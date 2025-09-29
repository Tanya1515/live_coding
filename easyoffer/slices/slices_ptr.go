package main

import (
  "fmt"
)

/*

Начиная с go 1.22: 

0 64
1 8  
2 32
3 16

До go 1.22: 

0 16
1 16
2 16
3 16

*/

// ptr
func main() {
  in := []int{64, 8, 32, 16}
  out := make([]*int, len(in))

  for idx, value := range in {
    out[idx] = &value
  }

  for idx, value := range out {
    fmt.Println(idx, *value) 
  }
}