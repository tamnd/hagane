package main

import "fmt"

func tryAssert(i interface{}) (val int, ok bool) {
	val, ok = i.(int)
	return
}

func main() {
	// comma-ok on nil interface → (0, false)
	v, ok := tryAssert(nil)
	fmt.Println(v, ok)

	// comma-ok on wrong type → (0, false)
	v2, ok2 := tryAssert("hello")
	fmt.Println(v2, ok2)

	// comma-ok on correct type → (42, true)
	v3, ok3 := tryAssert(42)
	fmt.Println(v3, ok3)
}
