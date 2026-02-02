package main

// import (
// 	"fmt"
// 	"sync"
// 	"time"
// )

// func main() {
// 	start := time.Now()

// 	// var ops atomic.Uint64
// 	var ops int64

// 	var wg sync.WaitGroup

// 	for range 50 {
// 		wg.Go(func() {
// 			for range 1000 {
// 				// ops.Add(1)
// 				ops += 1
// 			}
// 		})
// 	}

// 	wg.Wait()

// 	// fmt.Println("ops:", ops.Load())
// 	fmt.Println("ops:", ops)
// 	fmt.Println("time taken:", time.Since(start))
// }

// // --- OUTPUT ---
// // mayanktamakuwala@Mayanks-MacBook-Pro Homework-3 % go run atomic-counter.go
// // ops: 31652
// // time taken: 271.041µs
// // mayanktamakuwala@Mayanks-MacBook-Pro Homework-3 % go run atomic-counter.go
// // ops: 29285
// // time taken: 96.459µs
// // mayanktamakuwala@Mayanks-MacBook-Pro Homework-3 % go run atomic-counter.go
// // ops: 30902
// // time taken: 129.042µs
// // mayanktamakuwala@Mayanks-MacBook-Pro Homework-3 % go run atomic-counter.go
// // ops: 29259
// // time taken: 105.458µs
