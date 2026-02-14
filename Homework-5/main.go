package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

type Product struct {
	ProductID    int    `json:"product_id"`
	SKU          string `json:"sku"`
	Manufacturer string `json:"manufacturer"`
	CategoryID   int    `json:"category_id"`
	Weight       int    `json:"weight"`
	SomeOtherID  int    `json:"some_other_id"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

type Store struct {
	mu       sync.RWMutex
	products map[int]Product
}

func NewStore() *Store {
	return &Store{products: make(map[int]Product)}
}

func main() {
	store := NewStore()

	store.products[1] = Product{
		ProductID:    1,
		SKU:          "DELL-XPS-13",
		Manufacturer: "Dell",
		CategoryID:   2,
		Weight:       1200,
		SomeOtherID:  101,
	}
	store.products[2] = Product{
		ProductID:    2,
		SKU:          "SONY-WH1000XM4",
		Manufacturer: "Sony",
		CategoryID:   3,
		Weight:       250,
		SomeOtherID:  102,
	}
	store.products[3] = Product{
		ProductID:    3,
		SKU:          "LOGITECH-MX-MASTER",
		Manufacturer: "Logitech",
		CategoryID:   4,
		Weight:       135,
		SomeOtherID:  103,
	}
	store.products[4] = Product{
		ProductID:    4,
		SKU:          "SAMSUNG-970-EVO",
		Manufacturer: "Samsung",
		CategoryID:   5,
		Weight:       6,
		SomeOtherID:  104,
	}
	store.products[5] = Product{
		ProductID:    5,
		SKU:          "CORSAIR-K95-RGB",
		Manufacturer: "Corsair",
		CategoryID:   4,
		Weight:       1050,
		SomeOtherID:  105,
	}

	router := gin.Default()

	// GET /products/:productId
	router.GET("/products/:productId", func(c *gin.Context) {
		productID, ok := parsePositiveProductID(c)
		if !ok {
			return
		}

		store.mu.RLock()
		p, exists := store.products[productID]
		store.mu.RUnlock()

		if !exists {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Product not found", fmt.Sprintf("No product with id %d", productID))
			return
		}

		c.JSON(http.StatusOK, p)
	})

	// POST /products/:productId/details
	router.POST("/products/:productId/details", func(c *gin.Context) {
		productID, ok := parsePositiveProductID(c)
		if !ok {
			return
		}

		// Check product exists first -> 404 if not found (per spec)
		store.mu.RLock()
		_, exists := store.products[productID]
		store.mu.RUnlock()
		if !exists {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Product not found", fmt.Sprintf("No product with id %d", productID))
			return
		}

		var p Product
		if err := c.ShouldBindJSON(&p); err != nil {
			writeError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid input data", err.Error())
			return
		}

		// Validate required fields per api.yaml Product schema  [oai_citation:2‡api.yaml](sediment://file_00000000aeec720c982bd5e67da55631)
		if errMsg := validateProduct(p); errMsg != "" {
			writeError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid input data", errMsg)
			return
		}

		// Enforce body product_id matches path productId (good contract hygiene)
		if p.ProductID != productID {
			writeError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid input data", fmt.Sprintf("product_id in body must match productId in path: %d != %d", p.ProductID, productID))
			return
		}

		// Add/update details
		store.mu.Lock()
		store.products[productID] = p
		store.mu.Unlock()

		// 204 No Content
		c.Status(http.StatusNoContent)
	})

	// Run
	_ = router.Run(":8080")
}

func parsePositiveProductID(c *gin.Context) (int, bool) {
	raw := strings.TrimSpace(c.Param("productId"))
	id, err := strconv.Atoi(raw)
	if err != nil || id < 1 {
		writeError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid productId", "productId must be an integer >= 1")
		return 0, false
	}
	return id, true
}

func validateProduct(p Product) string {
	if p.ProductID < 1 {
		return "product_id is required and must be >= 1"
	}
	if strings.TrimSpace(p.SKU) == "" {
		return "sku is required and must be non-empty"
	}
	if len(p.SKU) > 100 {
		return "sku must be <= 100 characters"
	}
	if strings.TrimSpace(p.Manufacturer) == "" {
		return "manufacturer is required and must be non-empty"
	}
	if len(p.Manufacturer) > 200 {
		return "manufacturer must be <= 200 characters"
	}
	if p.CategoryID < 1 {
		return "category_id must be >= 1"
	}
	if p.Weight < 0 {
		return "weight must be >= 0"
	}
	if p.SomeOtherID < 1 {
		return "some_other_id must be >= 1"
	}
	return ""
}

func writeError(c *gin.Context, status int, code, message, details string) {
	c.JSON(status, ErrorResponse{
		Error:   code,
		Message: message,
		Details: details,
	})
}
