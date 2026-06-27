// wantpanic: nil pointer dereference
package main

type Greeter interface {
	Greet() string
}

func callGreet(g Greeter) string {
	return g.Greet()
}

func main() {
	var g Greeter
	_ = callGreet(g)
}
