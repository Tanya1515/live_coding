package main

import (
	"fmt"
	"time"
)

// блокировки не будет, поскольку мы итерируемся до конца массива и 
// вычитываем ровно то количество элементов, которое находится в массиве
// Но лучше все таки закрывать канал. 

// В комментариях указано решение лучше.
func ProcessData(data []int) {
	results := make(chan int, len(data))
	//var wg sync.WaitGroup

	for _, val := range data {
		//wg.Add(1)
		go func(x int) {
			//defer wg.Done()
			time.Sleep(1 * time.Second)
			results <- x * 2
		}(val)
	}

	// go func() {
	// 	wg.Wait()
	// 	close(results)
	// }()


	// for result := range results {
    //     fmt.Println(result)
    // }
	
	for i := 0; i < len(data); i++ {
		fmt.Println(<-results) // в рандомном порядке будет напечатано 2 4 6 8 10 12
	}
}

func main() {
	data := []int{1, 2, 3, 4, 5, 6}
	ProcessData(data)
}
