//go:build !m0

package main

import "fmt"

func main() {
	a := [4]string{"foo", "bar", "baz", "qux"}

	for i, v := range a {
		fmt.Println(i, v)
	}

	sum := 0
	nums := [5]int{1, 2, 3, 4, 5}
	for _, n := range nums {
		sum += n
	}
	fmt.Println(sum)
}
