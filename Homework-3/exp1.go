package main

import (
	"fmt"
	"time"
)

func exp1() {

	start := time.Now()

	add := func(x, y int32) int32 { return x + y }

	fmt.Println(add(3, 5))
	end := time.Now()
	fmt.Println("Execution Time:", end.Sub(start))
}

// func exp1() {
// 	start := time.Now()
// 	fmt.Println(add(3, 5))
// 	end := time.Now()
// 	fmt.Println("Execution Time:", end.Sub(start))
// }

// func add(x, y int32) int32 { return x + y }
