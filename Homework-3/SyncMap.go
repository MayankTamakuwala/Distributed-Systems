package main

// import (
// 	"fmt"
// 	"sync"
// 	"time"
// )

// func main() {
// 	start := time.Now()

// 	var wg sync.WaitGroup
// 	var m sync.Map
// 	wg.Add(50)

// 	for g := range 50 {
// 		go func(g int) {
// 			defer wg.Done()
// 			for i := range 1000 {
// 				m.Store((g*1000)+i, i)
// 			}
// 		}(g)
// 	}

// 	wg.Wait()

// 	mapLen := 0
// 	m.Range(func(_, _ any) bool {
// 		mapLen++
// 		return true
// 	})

// 	expected := 50 * 1000
// 	fmt.Printf("expected entries: %d\n", expected)
// 	fmt.Printf("actual len(m):   %d\n", mapLen)
// 	fmt.Printf("time taken: %v\n", time.Since(start))
// }
