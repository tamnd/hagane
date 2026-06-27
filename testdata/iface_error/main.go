package main

import "fmt"

type MyError struct {
	code    int
	message string
}

func (e MyError) Error() string {
	return fmt.Sprintf("error %d: %s", e.code, e.message)
}

func divide(a, b int) (int, error) {
	if b == 0 {
		return 0, MyError{code: 42, message: "division by zero"}
	}
	return a / b, nil
}

func main() {
	result, err := divide(10, 2)
	if err != nil {
		fmt.Println("Error:", err.Error())
	} else {
		fmt.Println("10 / 2 =", result)
	}

	result, err = divide(7, 0)
	if err != nil {
		fmt.Println("Error:", err.Error())
	} else {
		fmt.Println("7 / 0 =", result)
	}

	// type assertion on error interface
	_, err = divide(5, 0)
	if me, ok := err.(MyError); ok {
		fmt.Println("code:", me.code)
	}
}
