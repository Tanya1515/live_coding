package main

import (
	"encoding/json"
	"fmt"
)

type Base struct {
	Type string
}

type Derived struct {
	Base
	Value int
}

func main() {
	d := Derived{Base: Base{Type: "derived"}, Value: 42}

	b, _ := json.Marshal(d)

	fmt.Println(string(b))

	var base Base

	json.Unmarshal(b, &base)

	fmt.Printf("%+v\n", base)
}
