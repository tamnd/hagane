//go:build !m0

package main

import "fmt"

func main() {
	var s []int
	for i := 0; i < 12; i++ {
		s = append(s, i*i)
	}
	fmt.Println(len(s), s[11])

	t := []string{"a"}
	t = append(t, "b", "c", "d")
	fmt.Println(len(t), t[3])

	u := []int{1, 2}
	extra := []int{3, 4, 5}
	u = append(u, extra...)
	fmt.Println(len(u), u[4])
}
