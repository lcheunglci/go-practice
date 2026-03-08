package main

import (
	"fmt"
	"strings"
)

func main() {

	type score struct {
		name string
		score int 
	}

	// var name string = "Doe, Bob"
	// name := "Doe, Bob"
	// var score = 87
	// score := 87
	// name, score := "Doe, Bob", 87
	// students := []string{
	// 	 "Doe, Bob",
	//    "Jones, Jess",
	//    "Wonders, Alice",
	// }
	// scores := []int{87, 96, 64}
	// scores := map[string]int {
	// 	students[0]: 87,
	// 	students[1]: 96,
	// 	students[2]: 64,

	// }
	scores := []score {

		 {name: "Doe, Bob", score: 87},
	   {name: "Jones, Jess", score: 96},
	   {name: "Wonders, Alice", score: 64},

	}

	fmt.Println("Select score to print (1 - 3):")
	var option string
	fmt.Scanln(&option)

	fmt.Println("Student scores")
	fmt.Println(strings.Repeat("-", 14))
	// fmt.Println(students[0], scores[students[0]])
	// fmt.Println(students[1], scores[students[1]])
	// fmt.Println(students[2], scores[students[2]])

	var index int 
	switch option {
  case "1": // should use strconv package in production
		index = 0
	case "2":
		index = 1
	case "3":
		index = 2
	default:
		fmt.Println("Unknown option, defaulting to 1")
		index = 0
	}
	fmt.Println(scores[index].name, scores[index].score)
	// fmt.Println(scores[0].name, scores[0].score)
	// fmt.Println(scores[1].name, scores[1].score)
	// fmt.Println(scores[2].name, scores[2].score)
}