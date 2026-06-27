package main

import "fmt"

func main() {
	const n = 10000
	sieve := make([]bool, n+1)
	for i := 2; i <= n; i++ {
		sieve[i] = true
	}
	for i := 2; i*i <= n; i++ {
		if sieve[i] {
			for j := i * i; j <= n; j += i {
				sieve[j] = false
			}
		}
	}
	count := 0
	for i := 2; i <= n; i++ {
		if sieve[i] {
			count++
		}
	}
	fmt.Println(count)
}
