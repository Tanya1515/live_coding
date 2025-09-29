package main

func main() {
	s := "test"
	println(s[0])
	s[0] = "R" // ошибка компиляции
	println(s)
}
