package main

import "fmt"

func fib(n int) int {
	a, b := 0, 1
	for i := 0; i < n; i++ {
		a, b = b, a+b
	}
	return a
}

func main() {
	for i := 0; i <= 10; i++ {
		fmt.Println(fib(i))
	}
}
