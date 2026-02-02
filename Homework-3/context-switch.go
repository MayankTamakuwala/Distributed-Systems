package main

import (
	"fmt"
	"runtime"
	"time"
)

const iterations = 100000

func pingpong(ch1, ch2 chan int) {
	for i := 0; i < iterations; i++ {
		<-ch1
		ch2 <- 0
	}
}

func runTest() time.Duration {
	ch1 := make(chan int)
	ch2 := make(chan int)

	go pingpong(ch1, ch2)
	go pingpong(ch2, ch1)

	start := time.Now()

	ch1 <- 0
	<-ch1

	return time.Since(start)
}

func main() {
	// Single OS thread
	runtime.GOMAXPROCS(1)
	t1 := runTest()
	fmt.Println("GOMAXPROCS(1) time:", t1)

	// Default threads
	runtime.GOMAXPROCS(runtime.NumCPU())
	t2 := runTest()
	fmt.Println("Default GOMAXPROCS time:", t2)
}

// (14.151792ms + 38.196292ms + 12.794959ms + 26.574375ms)/4 = 22.9298545ms
// (19.542µs + 29.334µs + 15.958µs + 28µs)/4 = 23.2085µs
