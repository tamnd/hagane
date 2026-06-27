//go:build !m0

package main

import "fmt"

func main() {
	// string to []byte (must copy)
	s := "hello"
	b := []byte(s)
	b[0] = 'H'
	fmt.Println(string(b))
	fmt.Println(s) // original unchanged

	// []byte to string
	s2 := string([]byte{'g', 'o'})
	fmt.Println(s2)

	// string to []rune (UTF-8)
	r := []rune("héllo")
	fmt.Println(len(r))
	fmt.Println(r[1] == 'é')

	// []rune to string
	runes := []rune{'H', 'i', '!'}
	fmt.Println(string(runes))
}
