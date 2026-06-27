package main

import "fmt"

type Stringer interface {
	String() string
}

type Animal interface {
	Sound() string
	String() string
}

type Dog struct{ name string }

func (d Dog) Sound() string  { return "woof" }
func (d Dog) String() string { return "Dog:" + d.name }

func printStringer(s Stringer) {
	fmt.Println(s.String())
}

func main() {
	var a Animal = Dog{name: "Rex"}
	// ChangeInterface: Animal → Stringer
	printStringer(a)
	fmt.Println(a.Sound())
	fmt.Println(a.String())
}
