//go:build !m0 && !m1

package main

import (
	"errors"
	"fmt"
)

func divide(a, b int) (int, error) {
	if b == 0 {
		return 0, errors.New("division by zero")
	}
	return a / b, nil
}

func safeDivide(a, b int) (result int, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	return divide(a, b)
}

func main() {
	r, err := divide(10, 2)
	fmt.Println(r, err)

	r, err = divide(10, 0)
	fmt.Println(r, err)

	r, err = safeDivide(10, 2)
	fmt.Println(r, err)
}
