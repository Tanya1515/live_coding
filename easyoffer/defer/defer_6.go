package main

import (
	"fmt"
	"time"
)

// Когда main() завершается, ВСЕ работающие горутины принудительно останавливаются. 
// Точно будет напечатана 1, а Горутины с fmt.Println(2) и fmt.Println(3) могут не успеть выполниться 

func main() {
	defer fmt.Println(1) // ← Выполнится ПРИ ВЫХОДЕ из main
	time.Sleep(1 * time.Second)
	go fmt.Println(2) // ← Запуск горутины (может не успеть)
	go fmt.Println(3) // ← Запуск горутины (может не успеть)
}
