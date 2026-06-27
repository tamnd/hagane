package main

import "fmt"

func main() {
	s := make([]int, 100)
	for i := range s {
		s[i] = i + 1
	}
	sum := 0
	for _, v := range s {
		sum += v
	}
	fmt.Println(sum)
}
