package main

import "fmt"

func pow(base, exp int) int {
	result := 1
	for exp > 0 {
		if exp%2 == 1 {
			result *= base
		}
		base *= base
		exp /= 2
	}
	return result
}

func main() {
	fmt.Println(pow(2, 10))
	fmt.Println(pow(3, 5))
	fmt.Println(pow(10, 0))
}
