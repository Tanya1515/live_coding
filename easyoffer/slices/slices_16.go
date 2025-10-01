package main

import (
	"fmt"
)

type person struct {
	age int
}

/*

Далее представлены две функции, в каждой из которых создан либо обычный слайс, либо слайс указателей.

Такая разница, поскольку при append, если вся capacity слайса израсходована, то создается 
новый участок памяти, куда копируются все данные. Соответстсвенно, указатели меняются. 

*/

func sliceWithoutPointers() {
	fmt.Println("Slice without pointers:")
	list := []person{
		{age: 18},
		{age: 25},
	}

	alex := &list[1] // указатель на 2ой элемент в массиве
	alex.age++       // 26

	list = append(list, person{age: 30})
	alex.age++ // 27

	fmt.Println(list[1].age) // 26
	fmt.Println(alex.age)    // 27
}

func sliceWithPointers() {
	fmt.Println("Slice with pointers:")
	list := []*person{
		{age: 18},
		{age: 25},
	}

	alex := list[1] // указатель на 2ой элемент в массиве
	alex.age++      // 26

	list = append(list, &person{age: 30})
	alex.age++ // 27

	fmt.Println(list[1].age) // 27
	fmt.Println(alex.age)    // 27
}

// Map with pointers
// 32
// 32
// Map without pointers
// 32
// 30
func mapWithPointers() {
	fmt.Println("Map with pointers")
	mapPointers := make(map[int]*person, 0)

	mapPointers[0] = &person{age: 30}
	mapPointers[1] = &person{age: 25}

	personTest := mapPointers[0]
	personTest.age++

	for i := 2; i < 100; i++ {
		mapPointers[i] = &person{age: 30 + i}
	}

	personTest.age++

	fmt.Println(personTest.age)
	fmt.Println(mapPointers[0].age)

}

func mapWithoutPointers() {
	fmt.Println("Map without pointers")
	mapWithout := make(map[int]person, 0)

	mapWithout[0] = person{age: 30}
	mapWithout[1] = person{age: 25}

	personTest := mapWithout[0]
	personTest.age++

	for i := 2; i < 100; i++ {
		mapWithout[i] = person{age: 30 + i}
	}

	personTest.age++

	fmt.Println(personTest.age)
	fmt.Println(mapWithout[0].age)
}

func main() {
	sliceWithoutPointers()
	sliceWithPointers()
	mapWithPointers()
	mapWithoutPointers()
}
