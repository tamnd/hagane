package main

import "fmt"

type Shape interface {
	Area() float64
	Perimeter() float64
	Name() string
}

type Rectangle struct {
	width, height float64
}

func (r Rectangle) Area() float64      { return r.width * r.height }
func (r Rectangle) Perimeter() float64 { return 2 * (r.width + r.height) }
func (r Rectangle) Name() string       { return "Rectangle" }

type Circle struct {
	radius float64
}

func (c Circle) Area() float64      { return 3.14159 * c.radius * c.radius }
func (c Circle) Perimeter() float64 { return 2 * 3.14159 * c.radius }
func (c Circle) Name() string       { return "Circle" }

func printShape(s Shape) {
	fmt.Printf("%s: area=%.2f perimeter=%.2f\n", s.Name(), s.Area(), s.Perimeter())
}

func main() {
	shapes := []Shape{
		Rectangle{width: 3, height: 4},
		Circle{radius: 5},
		Rectangle{width: 10, height: 2},
	}
	for _, s := range shapes {
		printShape(s)
	}

	// type switch on Shape
	for _, s := range shapes {
		switch v := s.(type) {
		case Rectangle:
			fmt.Printf("rect %gx%g\n", v.width, v.height)
		case Circle:
			fmt.Printf("circle r=%g\n", v.radius)
		}
	}
}
