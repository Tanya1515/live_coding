package main

import "fmt"

func main() {
	contSlice := make([]string, 0, 2)
	contMap := map[struct{}]string{} // инициализация пустой мапы

	exampleB1(contSlice)
	exampleB2(contMap)

	// slice: []string{}
	// slice: map[struct {}]string{struct {}{}:"B"}
	fmt.Printf("slice: %#v\n", contSlice) // пустой slice, но если написать contSlice[:2] - будет напечатано "A" "B"
	fmt.Printf("slice: %#v\n", contMap) // struct{}{} -> "B"
}

// здесь копируется структура slice,
// а затем изменяется длина, но в slice из main 
// их не будет видно, поскольку предварительно 
// необходимо увеличить длину слайса
func exampleB1(cont []string) {
	cont = append(cont, "A")
	cont = append(cont, "B")
}

// здесь заполняется мапа
func exampleB2(cont map[struct{}]string) {
	cont[struct{}{}] = "A"
	cont[struct{}{}] = "B"
}
