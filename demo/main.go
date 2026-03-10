package main

import (
	"fmt"
	"strconv"
	"strings"
)

func main() {

	type score struct {
		name  string
		score int
	}

	scores := []score{}
	shouldContinue := true

	for shouldContinue == true {
		fmt.Println("1) Enter a score")
		fmt.Println("2) Print report")
		fmt.Println("q) Quit")
		fmt.Println()
		fmt.Println("Please select an option")

		var option string
		fmt.Scanln(&option)

		switch option {
		case "1":
			fmt.Println("Enter a student name and score")
			var name, rawScore string
			fmt.Scanln(&name, &rawScore)
			s, _ := strconv.Atoi(rawScore)
			scores = append(scores, score{name: name, score: s})

		case "2":
			fmt.Println("Student scores")
			fmt.Println(strings.Repeat("-", 14))
			for _, s := range scores {
				fmt.Println(s.name, s.score)
			}
		case "q":
			//tell the loop to stop
			shouldContinue = false
		}
	}

}
