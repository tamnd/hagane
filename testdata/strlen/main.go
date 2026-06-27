package main

import "fmt"

// byteLen counts the number of UTF-8 bytes in s by iterating byte by byte.
func byteLen(s string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		_ = s[i]
		n++
	}
	return n
}

func main() {
	words := []string{"hello", "world", "golang", "鋼", "日本語", "αβγ"}
	for _, w := range words {
		fmt.Printf("%s: %d bytes\n", w, byteLen(w))
	}
}
