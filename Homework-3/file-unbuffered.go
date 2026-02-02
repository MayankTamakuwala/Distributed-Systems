package main

// import (
// 	"fmt"
// 	"os"
// 	"time"
// )

// func main() {
// 	f, err := os.Create("output.txt")
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer f.Close()

// 	start := time.Now()
// 	for i := 0; i < 50000; i++ {
// 		f.WriteString("Hello Distributed Systems\n")
// 	}

// 	elapsed := time.Since(start)

// 	fmt.Println("Unbuffered write finished")
// 	fmt.Println("Time taken:", elapsed)
// }

// // 106.417885ms
