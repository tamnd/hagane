//go:build !m0 && !m1

package main

import "fmt"

func main() {
outer:
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			if i+j == 4 {
				break outer
			}
			fmt.Printf("%d+%d=%d\n", i, j, i+j)
		}
	}

loop:
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if j == 1 {
				continue loop
			}
			fmt.Println(i, j)
		}
	}
}
