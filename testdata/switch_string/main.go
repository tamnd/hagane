//go:build !m0 && !m1

package main

import "fmt"

func grade(score int) string {
	switch {
	case score >= 90:
		return "A"
	case score >= 80:
		return "B"
	case score >= 70:
		return "C"
	default:
		return "F"
	}
}

func dayType(day string) string {
	switch day {
	case "Saturday", "Sunday":
		return "weekend"
	case "Monday", "Tuesday", "Wednesday", "Thursday", "Friday":
		return "weekday"
	default:
		return "unknown"
	}
}

func main() {
	for _, s := range []int{95, 85, 72, 55} {
		fmt.Println(grade(s))
	}
	for _, d := range []string{"Monday", "Saturday", "Holiday"} {
		fmt.Println(dayType(d))
	}
}
