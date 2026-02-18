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
	name, score := "Doe, Bob", 87

	fmt.Println("Student scores")
	fmt.Println(strings.Repeat("-", 14))
	fmt.Println(name, score)
}