package main

import "fmt"

type Mat4 [4][4]float64

func mul(a, b Mat4) Mat4 {
	var c Mat4
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			for k := 0; k < 4; k++ {
				c[i][j] += a[i][k] * b[k][j]
			}
		}
	}
	return c
}

func main() {
	a := Mat4{
		{1, 2, 3, 4},
		{5, 6, 7, 8},
		{9, 10, 11, 12},
		{13, 14, 15, 16},
	}
	b := Mat4{
		{1, 0, 0, 0},
		{0, 1, 0, 0},
		{0, 0, 1, 0},
		{0, 0, 0, 1},
	}
	c := mul(a, b)
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			if j > 0 {
				fmt.Print(" ")
			}
			fmt.Printf("%.0f", c[i][j])
		}
		fmt.Println()
	}
}
