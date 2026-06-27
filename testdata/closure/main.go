//go:build !m0 && !m1

package main

import "fmt"

func makeCounter() func() int {
	n := 0
	return func() int {
		n++
		return n
	}
}

func main() {
	c := makeCounter()
	fmt.Println(c())
	fmt.Println(c())
	fmt.Println(c())

	d := makeCounter()
	fmt.Println(d())
}
