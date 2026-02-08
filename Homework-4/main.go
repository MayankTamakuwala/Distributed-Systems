package main

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

func main() {
	bucket := "mayank-hw4-mapreduce-2026"
	splitterURL := fmt.Sprintf("http://44.245.170.234:8080/split?bucket=%s&key=input.txt&chunks=3", bucket)
	mapperURLs := []string{
		fmt.Sprintf("http://34.211.52.40:8080/map?bucket=%s&key=chunks/chunk0.txt", bucket),
		fmt.Sprintf("http://34.223.66.54:8080/map?bucket=%s&key=chunks/chunk1.txt", bucket),
		fmt.Sprintf("http://44.249.163.29:8080/map?bucket=%s&key=chunks/chunk2.txt", bucket),
	}
	reducerURL := fmt.Sprintf("http://35.90.78.24:8080/reduce?bucket=%s&key=maps/chunk0.json&key=maps/chunk1.json&key=maps/chunk2.json", bucket)

	// Splitter
	start := time.Now()
	resp, err := http.Get(splitterURL)
	if err != nil {
		fmt.Println("splitter request error:", err)
	} else {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		fmt.Println("Splitter URL:", splitterURL)
		fmt.Println("Splitter Status:", resp.StatusCode)
		fmt.Println("Splitter Response:", string(body))
		fmt.Println("Splitter Time:", time.Since(start))
		fmt.Println()
	}

	// Mappers in parallel
	mapStart := time.Now()
	var wg sync.WaitGroup
	for i, u := range mapperURLs {
		wg.Add(1)
		go func(i int, url string) {
			defer wg.Done()
			s := time.Now()
			r, e := http.Get(url)
			if e != nil {
				fmt.Println("mapper request error:", e)
				return
			}
			b, _ := io.ReadAll(r.Body)
			r.Body.Close()
			fmt.Printf("Mapper %d URL: %s\n", i+1, url)
			fmt.Printf("Mapper %d Status: %d\n", i+1, r.StatusCode)
			fmt.Printf("Mapper %d Response: %s\n", i+1, string(b))
			fmt.Printf("Mapper %d Time: %s\n", i+1, time.Since(s))
			fmt.Println()
		}(i, u)
	}
	wg.Wait()
	fmt.Println("Total Mapper Phase Time:", time.Since(mapStart))
	fmt.Println()

	// Reducer
	reduceStart := time.Now()
	reduceResp, reduceErr := http.Get(reducerURL)
	if reduceErr != nil {
		fmt.Println("reducer request error:", reduceErr)
		return
	}
	reduceBody, _ := io.ReadAll(reduceResp.Body)
	reduceResp.Body.Close()
	fmt.Println("Reducer URL:", reducerURL)
	fmt.Println("Reducer Status:", reduceResp.StatusCode)
	fmt.Println("Reducer Response:", string(reduceBody))
	fmt.Println("Reducer Time:", time.Since(reduceStart))
}
