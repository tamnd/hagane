//go:build !m0

package main

import "fmt"

func main() {
	s := []int{10, 20, 30, 40}

	for i, v := range s {
		fmt.Println(i, v)
	}

	for i := range s {
		fmt.Println(i)
	}

	sum := 0
	for _, v := range s {
		sum += v
	}
	fmt.Println(sum)
}
