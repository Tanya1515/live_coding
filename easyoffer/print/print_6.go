package main

import (
	"fmt"
	"time"
)

const numRequests = 1000

var count int

func networkRequest() {
	time.Sleep(time.Millisecond)

	count++
}

func main() {
	defer timer()()

	for i := 0; i < numRequests; i++ {
		networkRequest()
	}
}

func timer() func() {
	start := time.Now()

	return func() {
		fmt.Printf("count %v took %v\n", count, time.Since(start)) // count 1000 took 1.280122523s
	}
}
