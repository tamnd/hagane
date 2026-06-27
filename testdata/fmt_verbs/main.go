package main

import "fmt"

func main() {
	// %T verb: type name of the boxed value
	fmt.Printf("%T\n", 42)
	fmt.Printf("%T\n", 3.14)
	fmt.Printf("%T\n", "hello")
	fmt.Printf("%T\n", true)

	// %T via Sprintf
	s := fmt.Sprintf("type is %T", 100)
	fmt.Println(s)
}
