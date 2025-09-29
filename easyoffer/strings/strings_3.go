package main

import "strings"

/*

Есть сообщения из соцсети, например:
"Я работаю в Гугле :-)))"
"везет :-) а я туда собеседование завалил:-(("
"лол:)"
"Ааааа!!!!! :-))(())"

Хочется удалить из них смайлики, подпадающие под регулярку ":-\)+|:-\(+" за линейное время. То есть, сделать так:
"Я работаю в Гугле "
"везет  а я туда собеседование завалил"
"лол:)"
"Ааааа!!!!! (())"

Специально обрабатывать вложенные смайлики нельзя. Так, :-:-)))((( должно превратиться в :-(((, но не в пустую строку.

*/

func deleteSmiles(str string) string {
	var result strings.Builder

	smileFine := 0

	i := 0

	var strRune []rune
	strRune = []rune(str)

	for i < len(strRune) {
		if smileFine == 1 && strRune[i] == ')' {
			i++
			continue
		}
		if smileFine == 2 && strRune[i] == '(' {
			i++
			continue
		}
		if i+2 < len(strRune) && string(strRune[i]+strRune[i+1]+strRune[i+2]) == ":-)" {
			smileFine = 1
			i += 3
			continue
		}
		if i+2 < len(strRune) && string(strRune[i]+strRune[i+1]+strRune[i+2]) == ":-(" {
			smileFine = 2
			i += 3
			continue
		}
		smileFine = 0
		result.WriteRune(strRune[i])
		i++
	}

	return result.String()

}
