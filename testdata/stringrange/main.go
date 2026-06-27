package main

import "fmt"

func runeCount(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}

func main() {
	s := "hello"
	for i, r := range s {
		fmt.Println(i, r)
	}
	fmt.Println(runeCount("hello"))
	fmt.Println(runeCount("日本語"))
}
