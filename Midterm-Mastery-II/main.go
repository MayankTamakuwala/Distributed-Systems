package main

import (
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type Product struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Brand       string `json:"brand"`
}

type SearchResponse struct {
	Products   []Product `json:"products"`
	TotalFound int       `json:"total_found"`
	SearchTime string    `json:"search_time"`
}

var products sync.Map

func main() {
	// Seed randomness so failures are actually random across runs
	rand.Seed(time.Now().UnixNano())

	generateProducts(100_000)

	router := gin.Default()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	router.GET("/products/search", func(c *gin.Context) {
		start := time.Now()

		// ----- Step III: simulate downstream dependency -----
		timeout := 200 * time.Millisecond

		// var depErr = fakeExternalService()
		var depErr = fakeExternalServiceWithTimeout(timeout)

		if depErr != nil {
			// Controlled failure: we return fast instead of hanging
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":  "dependency failed",
				"detail": depErr.Error(),
			})
			return
		}
		// -----------------------------------------------

		q := strings.TrimSpace(c.Query("q"))
		if q == "" {
			c.JSON(http.StatusOK, SearchResponse{Products: []Product{}, TotalFound: 0, SearchTime: "0ms"})
			return
		}

		qLower := strings.ToLower(q)
		results := make([]Product, 0, 20)
		totalFound := 0
		checked := 0

		products.Range(func(_, v any) bool {
			p, ok := v.(Product)
			if !ok {
				return true
			}
			checked++

			nameLower := strings.ToLower(p.Name)
			catLower := strings.ToLower(p.Category)
			if strings.Contains(nameLower, qLower) || strings.Contains(catLower, qLower) {
				totalFound++
				if len(results) < 20 {
					results = append(results, p)
				}
			}

			return checked < 100
		})

		elapsed := time.Since(start)
		c.JSON(http.StatusOK, SearchResponse{
			Products:   results,
			TotalFound: totalFound,
			SearchTime: fmt.Sprintf("%.2fms", float64(elapsed.Microseconds())/1000),
		})
	})

	_ = router.Run(":8080")
}

// this is the "external service" that we depend on, which is unreliable
// and can be slow it simulates a 30% chance of being very slow (2 seconds)
// and failing, and otherwise responds quickly (10ms)
// fake downstream service (BROKEN behavior)
func fakeExternalService() error {
	// 30% chance to be very slow
	if rand.Intn(100) < 30 {
		time.Sleep(2 * time.Second)
		return errors.New("downstream timeout (simulated)")
	}

	// otherwise respond quickly
	time.Sleep(10 * time.Millisecond)
	return nil
}

// fail-fast wrapper (FIXED behavior)
func fakeExternalServiceWithTimeout(timeout time.Duration) error {
	done := make(chan error, 1)

	go func() {
		done <- fakeExternalService()
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		return errors.New("dependency timed out fast (fail-fast)")
	}
}

func generateProducts(n int) {
	brands := []string{"Alpha", "Nimbus", "Orion", "Vertex", "Pulse", "Apex", "Nova", "Summit", "Zephyr", "Titan", "Eclipse", "Vortex", "Zenith", "Horizon", "Blaze", "Crest"}
	categories := []string{"Electronics", "Home", "Books", "Clothing", "Sports", "Beauty", "Toys", "Grocery", "Furniture", "Automotive", "Garden", "Health", "Music", "Office", "Pets", "Travel"}
	descriptions := []string{
		"High quality and reliable.",
		"Designed for everyday use.",
		"New and improved version.",
		"Customer favorite product.",
		"Lightweight and durable.",
		"Limited edition release.",
		"Eco-friendly and sustainable.",
		"Premium materials throughout.",
		"Award-winning design.",
		"Best value in its class.",
		"Highly rated by experts.",
		"Perfect for professionals.",
		"Compact and easy to store.",
		"Backed by a lifetime warranty.",
		"Fast and efficient performance.",
		"Trusted by millions worldwide.",
	}

	for id := 1; id <= n; id++ {
		brand := brands[id%len(brands)]
		category := categories[id%len(categories)]
		desc := descriptions[id%len(descriptions)]

		p := Product{
			ID:          id,
			Brand:       brand,
			Category:    category,
			Name:        "Product " + brand + " " + strconv.Itoa(id),
			Description: desc,
		}
		products.Store(id, p)
	}
}
