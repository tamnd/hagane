//go:build !m0 && !m1

package main

import "fmt"

func main() {
	n := 0
	for {
		if n >= 3 {
			break
		}
		fmt.Println(n)
		n++
	}
	fmt.Println("done")
}
