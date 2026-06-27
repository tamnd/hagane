package main

import "fmt"

func sum(s []int) int {
	total := 0
	for i := 0; i < len(s); i++ {
		total += s[i]
	}
	return total
}

func main() {
	s := []int{1, 2, 3, 4, 5}
	fmt.Println(sum(s))
}
