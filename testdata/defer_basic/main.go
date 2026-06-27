//go:build !m0 && !m1

package main

import "fmt"

func countdown() {
	defer fmt.Println("done")
	for i := 3; i >= 1; i-- {
		defer fmt.Println(i)
	}
	fmt.Println("counting")
}

func withReturn() int {
	x := 0
	defer func() { x++ }()
	x = 10
	return x
}

func main() {
	countdown()
	fmt.Println(withReturn())
}
