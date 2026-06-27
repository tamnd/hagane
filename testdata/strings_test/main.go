package main

import "fmt"

func main() {
	a := "hello"
	b := "world"
	c := a + ", " + b + "!"
	fmt.Println(c)
	fmt.Println(len(c))
	fmt.Println(a == "hello")
	fmt.Println(a < b)
}
