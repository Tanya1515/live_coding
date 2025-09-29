package main

import "strings"

/*

Есть массив строк, необходимо вывести каждую 2ую строку и разделить их пробелом и запятой.

["abc" "def" "kjh"] -> "def"

["abc" "def" "kjh" "hgf"] -> "def", "hgf"

*/

func makeString(strs []string) string {

	var result strings.Builder
	for i := 0; i < len(strs); i++ {
		if i%2 == 0 {
			continue
		}
		result.WriteString(strs[i])
		if i != len(strs)-1 || i != len(strs)-2 {
			result.WriteString(",")
			result.WriteString(" ")
		}
	}

	return result.String()
}
