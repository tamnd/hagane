//go:build !m0 && !m1

package main

import "fmt"

func riskyWork() {
	defer fmt.Println("riskyWork: defer 1")
	defer fmt.Println("riskyWork: defer 2")
	defer fmt.Println("riskyWork: defer 3")
	panic("something went wrong")
}

func safe() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("recovered:", r)
		}
	}()
	riskyWork()
	fmt.Println("this should not print")
}

func main() {
	safe()
	fmt.Println("main continues")
}
