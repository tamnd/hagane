//go:build !m0 && !m1

package main

import "fmt"

func double(n int) (result int) {
	defer func() { result *= 2 }()
	result = n
	return
}

func withError() (n int, err error) {
	defer func() {
		if err != nil {
			n = -1
		}
	}()
	n = 42
	return
}

func main() {
	fmt.Println(double(7))
	n, err := withError()
	fmt.Println(n, err)
}
