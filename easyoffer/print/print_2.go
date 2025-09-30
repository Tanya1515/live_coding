package main

import "fmt"

type Person struct {
	Name string
}

func (person Person) SetName(newName string) {
	person.Name = person.Name + newName
}

func main() {
	person := Person{
		Name: "Bob",
	}

	fmt.Println(person.Name) // Bob

	person.SetName("Alice") // здесь будет передаваться и модифицироваться копия структуры

	fmt.Println(person.Name) // Bob

}
