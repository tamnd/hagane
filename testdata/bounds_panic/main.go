// wantpanic: index out of range
package main

func main() {
	s := []int{1, 2, 3}
	_ = s[5]
}
