package main

import (
	"fmt"
	"strconv"
)

func main() {

	type score struct {
		name string
		score int 
	}
	
	scores := []score {}

	for {
		fmt.Println("Enter a student name and score")
		var name, rawScore string;
		fmt.Scanln(&name, &rawScore)
		s, _ := strconv.Atoi(rawScore)
		scores = append(scores, score{name: name, score: s})
	}

	// fmt.Println("Select score to print (1 - 3):")
	// var option string
	// fmt.Scanln(&option)

	// fmt.Println("Student scores")
	// fmt.Println(strings.Repeat("-", 14))
	// // fmt.Println(students[0], scores[students[0]])
	// // fmt.Println(students[1], scores[students[1]])
	// // fmt.Println(students[2], scores[students[2]])

	// var index int 
	// switch option {
  // case "1": // should use strconv package in production
	// 	index = 0
	// case "2":
	// 	index = 1
	// case "3":
	// 	index = 2
	// default:
	// 	fmt.Println("Unknown option, defaulting to 1")
	// 	index = 0
	// }
	// fmt.Println(scores[index].name, scores[index].score)
	// fmt.Println(scores[0].name, scores[0].score)
	// fmt.Println(scores[1].name, scores[1].score)
	// fmt.Println(scores[2].name, scores[2].score)
}