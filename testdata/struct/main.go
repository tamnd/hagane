//go:build !m0

package main

import (
	"fmt"
	"math"
)

type Point struct {
	X, Y float64
}

func (p Point) Distance(q Point) float64 {
	dx := p.X - q.X
	dy := p.Y - q.Y
	return math.Sqrt(dx*dx + dy*dy)
}

func main() {
	a := Point{0, 0}
	b := Point{3, 4}
	fmt.Printf("%.6f\n", a.Distance(b))
}
