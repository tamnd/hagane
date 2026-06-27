package main

import "fmt"

func main() {
	m := map[int]int{1: 10, 2: 20, 3: 30}
	sum := 0
	for k, v := range m {
		sum += k * v
	}
	// sum should be 1*10 + 2*20 + 3*30 = 10+40+90 = 140
	fmt.Println(sum)
}
