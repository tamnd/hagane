package main

import "fmt"

func collatzLen(n int) int {
	steps := 0
	for n != 1 {
		if n%2 == 0 {
			n /= 2
		} else {
			n = 3*n + 1
		}
		steps++
	}
	return steps
}

func main() {
	for _, n := range []int{1, 6, 27, 100} {
		fmt.Println(collatzLen(n))
	}
}
