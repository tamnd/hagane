package main

import "fmt"

type Counter interface {
	Inc()
	Value() int
}

type MyCounter struct {
	n int
}

func (c *MyCounter) Inc()       { c.n++ }
func (c *MyCounter) Value() int { return c.n }

func run(c Counter) {
	c.Inc()
	c.Inc()
	c.Inc()
	fmt.Println(c.Value())
}

func main() {
	c := &MyCounter{}
	run(c)
	fmt.Println(c.Value())
}
