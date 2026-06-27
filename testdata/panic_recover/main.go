//go:build !m0 && !m1

package main

import "fmt"

func safeDivide(a, b int) (result int, caught bool) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("recovered:", r)
			caught = true
		}
	}()
	if b == 0 {
		panic("division by zero")
	}
	return a / b, false
}

func mustPositive(n int) int {
	if n < 0 {
		panic(fmt.Sprintf("negative value: %d", n))
	}
	return n
}

func safePositive(n int) (v int, ok bool) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("caught:", r)
			ok = false
		}
	}()
	v = mustPositive(n)
	ok = true
	return
}

func main() {
	r, ok := safeDivide(10, 2)
	fmt.Println(r, ok)

	r, ok = safeDivide(10, 0)
	fmt.Println(r, ok)

	v, ok := safePositive(5)
	fmt.Println(v, ok)

	v, ok = safePositive(-3)
	fmt.Println(v, ok)
}
