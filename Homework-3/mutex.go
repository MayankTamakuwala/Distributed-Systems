package main

// import (
// 	"fmt"
// 	"sync"
// 	"time"
// )

// func main() {
// 	start := time.Now()
// 	m := make(map[int]int)

// 	var wg sync.WaitGroup
// 	var mutex sync.Mutex
// 	wg.Add(50)

// 	for g := range 50 {
// 		go func(g int) {
// 			defer wg.Done()
// 			for i := range 1000 {
// 				mutex.Lock()
// 				m[(g*1000)+i] = i
// 				mutex.Unlock()
// 			}
// 		}(g)
// 	}

// 	wg.Wait()

// 	expected := 50 * 1000
// 	fmt.Printf("expected entries: %d\n", expected)
// 	fmt.Printf("actual len(m):   %d\n", len(m))
// 	fmt.Printf("time taken: %v\n", time.Since(start))
// }
