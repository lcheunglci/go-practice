package main

import (
	"fmt"
	"strings"
)

func main() {
	// var name string = "Doe, Bob"
	// name := "Doe, Bob"
	// var score = 87
	// score := 87
	// name, score := "Doe, Bob", 87
	students := []string{"Doe, Bob",
	   "Jones, Jess",
	   "Wonders, Alice",
	}
	scores := []int{87, 96, 64}

	fmt.Println("Student scores")
	fmt.Println(strings.Repeat("-", 14))
	fmt.Println(students[0], scores[0])
	fmt.Println(students[1], scores[1])
	fmt.Println(students[2], scores[2])
}