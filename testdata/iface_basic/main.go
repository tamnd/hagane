package main

import "fmt"

type Greeter interface {
	Greet() string
}

type English struct{ name string }
type Spanish struct{ name string }

func (e English) Greet() string { return "Hello, " + e.name }
func (s Spanish) Greet() string { return "Hola, " + s.name }

func greet(g Greeter) {
	fmt.Println(g.Greet())
}

func main() {
	greet(English{"World"})
	greet(Spanish{"Mundo"})

	var g Greeter = English{"Go"}
	fmt.Println(g.Greet())

	// comma-ok type assertion
	if e, ok := g.(English); ok {
		fmt.Println("English:", e.name)
	}
	if _, ok := g.(Spanish); !ok {
		fmt.Println("not Spanish")
	}

	// type switch
	switch v := g.(type) {
	case English:
		fmt.Println("switch English:", v.name)
	case Spanish:
		fmt.Println("switch Spanish:", v.name)
	default:
		fmt.Println("unknown")
	}
}
