//go:build !m0

package main

import "fmt"

func main() {
	src := []int{1, 2, 3, 4, 5}
	dst := make([]int, 3)
	n := copy(dst, src)
	fmt.Println(n, dst[0], dst[1], dst[2])

	// copy into larger dst
	dst2 := make([]int, 10)
	n2 := copy(dst2, src)
	fmt.Println(n2)

	// copy bytes
	b := make([]byte, 4)
	n3 := copy(b, []byte("hello"))
	fmt.Println(n3, string(b))
}
