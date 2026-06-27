package main

import "fmt"

func main() {
	m := map[string]int{}
	words := []string{"apple", "banana", "apple", "cherry", "banana", "apple"}
	for _, w := range words {
		m[w]++
	}
	fmt.Println(m["apple"])
	fmt.Println(m["banana"])
	fmt.Println(m["cherry"])
	fmt.Println(m["missing"])

	v, ok := m["apple"]
	fmt.Println(v, ok)
	v, ok = m["missing"]
	fmt.Println(v, ok)

	delete(m, "banana")
	fmt.Println(m["banana"])
	fmt.Println(len(m))
}
