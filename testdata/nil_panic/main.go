// wantpanic: nil pointer dereference
package main

func main() {
	var p *int
	_ = *p
}
