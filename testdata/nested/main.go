package main

import "fmt"

func main() {
	for i := 1; i <= 3; i++ {
		for j := 1; j <= 3; j++ {
			if i == j {
				fmt.Printf("%d==%d\n", i, j)
			} else if i > j {
				fmt.Printf("%d>%d\n", i, j)
			} else {
				fmt.Printf("%d<%d\n", i, j)
			}
		}
	}
}
