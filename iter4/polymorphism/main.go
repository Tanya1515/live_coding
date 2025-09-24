package main

import (
	"encoding/json"
	"fmt"
)

type Base struct {
	Type string
}

// встраивание
type Derived struct {
	Base
	Value int
}

func main() {
	d := Derived{Base: Base{Type: "derived"}, Value: 42}

	b, _ := json.Marshal(d)

	fmt.Println(string(b)) // {"Type":"derived","Value":42}

	var base Base

	json.Unmarshal(b, &base) // Unmarshal падает только на невалидном json

	fmt.Printf("%+v\n", base) // {Type:derived}, поскольку в json присустсвует поле Type и значение derived
}
