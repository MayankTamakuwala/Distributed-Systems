package main

// import (
// 	"bufio"
// 	"fmt"
// 	"os"
// 	"time"
// )

// const lines = 50000

// func main() {
// 	start := time.Now()

// 	f, err := os.Open("output.txt")
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer f.Close()

// 	writer := bufio.NewWriter(f)

// 	for i := 0; i < lines; i++ {
// 		writer.WriteString("Hello Distributed Systems\n")
// 	}

// 	writer.Flush()

// 	elapsed := time.Since(start)

// 	fmt.Println("Buffered write finished")
// 	fmt.Println("Time taken:", elapsed)
// }

// // (423.584µs + 460.25µs + 173.916µs + 181.291µs)/4 = 309.260875µs
