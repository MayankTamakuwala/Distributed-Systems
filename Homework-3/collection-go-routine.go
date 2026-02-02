package main

// import (
// 	"fmt"
// 	"sync"
// )

// func main() {
// 	m := make(map[int]int)

// 	var wg sync.WaitGroup
// 	wg.Add(50)

// 	for g := range 50 {
// 		go func(g int) {
// 			defer wg.Done()
// 			for i := range 1000 {
// 				m[(g*1000)+i] = i
// 			}
// 		}(g)
// 	}

// 	wg.Wait()

// 	expected := 50 * 1000
// 	fmt.Printf("expected entries: %d\n", expected)
// 	fmt.Printf("actual len(m):   %d\n", len(m))
// }
