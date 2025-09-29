package main

import "fmt"

// Пример использования type assertion

func main() {
	var i interface{} = "привет" // i содержит строку

	// Первая форма (приводит к panic при ошибке)
	// str := i.(string) // str теперь "привет"
	// fmt.Println(str)

	// Вторая форма (безопасная)
	str, ok := i.(string)
	if ok {
		fmt.Printf("Успешно! Значение: %s\n", str) // Выведет: Успешно! Значение: привет
	} else {
		fmt.Println("Не удалось получить значение как string")
	}

	// Попробуем получить значение другого типа
	num, ok := i.(int)
	if ok {
		fmt.Printf("Число: %d\n", num)
	} else {
		fmt.Println("Не удалось получить значение как int") // Выведет: Не удалось получить значение как int
	}

}
