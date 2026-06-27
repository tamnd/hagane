//go:build !m0 && !m1

package main

import "fmt"

func main() {
	for i := 1; i <= 4; i++ {
		switch i {
		case 1:
			fmt.Println("one")
			fallthrough
		case 2:
			fmt.Println("two or after one")
		case 3:
			fmt.Println("three")
		default:
			fmt.Println("other")
		}
	}
}
