package main

import "fmt"

func minmax(s []int) (int, int) {
	if len(s) == 0 {
		return 0, 0
	}
	lo, hi := s[0], s[0]
	for i := 1; i < len(s); i++ {
		if s[i] < lo {
			lo = s[i]
		}
		if s[i] > hi {
			hi = s[i]
		}
	}
	return lo, hi
}

func main() {
	s := []int{3, 1, 4, 1, 5, 9, 2, 6, 5, 3}
	lo, hi := minmax(s)
	fmt.Println(lo, hi)
}
