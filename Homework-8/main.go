package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"homework8/store"

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

// in-memory product store (carried over from HW5)
type ProductStore struct {
	mu       sync.RWMutex
	products map[int]Product
}

func NewProductStore() *ProductStore {
	return &ProductStore{products: make(map[int]Product)}
}

func main() {
	productStore := NewProductStore()

	// same seed data from previous homework
	productStore.products[1] = Product{ProductID: 1, SKU: "DELL-XPS-13", Manufacturer: "Dell", CategoryID: 2, Weight: 1200, SomeOtherID: 101}
	productStore.products[2] = Product{ProductID: 2, SKU: "SONY-WH1000XM4", Manufacturer: "Sony", CategoryID: 3, Weight: 250, SomeOtherID: 102}
	productStore.products[3] = Product{ProductID: 3, SKU: "LOGITECH-MX-MASTER", Manufacturer: "Logitech", CategoryID: 4, Weight: 135, SomeOtherID: 103}
	productStore.products[4] = Product{ProductID: 4, SKU: "SAMSUNG-970-EVO", Manufacturer: "Samsung", CategoryID: 5, Weight: 6, SomeOtherID: 104}
	productStore.products[5] = Product{ProductID: 5, SKU: "CORSAIR-K95-RGB", Manufacturer: "Corsair", CategoryID: 4, Weight: 1050, SomeOtherID: 105}

	// pick mysql or dynamodb based on env var
	cartStore, err := initCartStore()
	if err != nil {
		log.Fatalf("cart store init failed: %v", err)
	}
	defer cartStore.Close()

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// --- products (same as HW5) ---

	r.GET("/products/:productId", func(c *gin.Context) {
		id, ok := parsePositiveInt(c, "productId")
		if !ok {
			return
		}
		productStore.mu.RLock()
		p, exists := productStore.products[id]
		productStore.mu.RUnlock()

		if !exists {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Product not found", fmt.Sprintf("No product with id %d", id))
			return
		}
		c.JSON(http.StatusOK, p)
	})

	r.POST("/products/:productId/details", func(c *gin.Context) {
		id, ok := parsePositiveInt(c, "productId")
		if !ok {
			return
		}
		productStore.mu.RLock()
		_, exists := productStore.products[id]
		productStore.mu.RUnlock()
		if !exists {
			writeError(c, http.StatusNotFound, "NOT_FOUND", "Product not found", fmt.Sprintf("No product with id %d", id))
			return
		}

		var p Product
		if err := c.ShouldBindJSON(&p); err != nil {
			writeError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid input data", err.Error())
			return
		}
		if msg := validateProduct(p); msg != "" {
			writeError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid input data", msg)
			return
		}
		if p.ProductID != id {
			writeError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid input data",
				fmt.Sprintf("product_id in body (%d) doesn't match path (%d)", p.ProductID, id))
			return
		}

		productStore.mu.Lock()
		productStore.products[id] = p
		productStore.mu.Unlock()
		c.Status(http.StatusNoContent)
	})

	// --- shopping carts (new for HW8) ---

	r.POST("/shopping-carts", func(c *gin.Context) {
		var req struct {
			CustomerID int `json:"customer_id"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid input data", err.Error())
			return
		}
		if req.CustomerID < 1 {
			writeError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid input data", "customer_id must be >= 1")
			return
		}

		cartID, err := cartStore.CreateCart(c.Request.Context(), req.CustomerID)
		if err != nil {
			writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create cart", err.Error())
			return
		}
		c.JSON(http.StatusCreated, gin.H{"shopping_cart_id": cartID})
	})

	r.GET("/shopping-carts/:shoppingCartId", func(c *gin.Context) {
		cartID, ok := parsePositiveInt(c, "shoppingCartId")
		if !ok {
			return
		}
		cart, err := cartStore.GetCart(c.Request.Context(), cartID)
		if err != nil {
			if errors.Is(err, store.ErrCartNotFound) {
				writeError(c, http.StatusNotFound, "NOT_FOUND", "Shopping cart not found",
					fmt.Sprintf("No cart with id %d", cartID))
				return
			}
			writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve cart", err.Error())
			return
		}
		c.JSON(http.StatusOK, cart)
	})

	r.POST("/shopping-carts/:shoppingCartId/items", func(c *gin.Context) {
		cartID, ok := parsePositiveInt(c, "shoppingCartId")
		if !ok {
			return
		}
		var req struct {
			ProductID int `json:"product_id"`
			Quantity  int `json:"quantity"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid input data", err.Error())
			return
		}
		if req.ProductID < 1 {
			writeError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid input data", "product_id must be >= 1")
			return
		}
		if req.Quantity < 1 {
			writeError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid input data", "quantity must be >= 1")
			return
		}

		if err := cartStore.AddItem(c.Request.Context(), cartID, req.ProductID, req.Quantity); err != nil {
			if errors.Is(err, store.ErrCartNotFound) {
				writeError(c, http.StatusNotFound, "NOT_FOUND", "Shopping cart not found",
					fmt.Sprintf("No cart with id %d", cartID))
				return
			}
			writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to add item", err.Error())
			return
		}
		c.Status(http.StatusNoContent)
	})

	r.Run(":8080")
}

func initCartStore() (store.CartStore, error) {
	switch os.Getenv("DB_TYPE") {
	case "mysql":
		dsn := os.Getenv("MYSQL_DSN")
		if dsn == "" {
			return nil, fmt.Errorf("MYSQL_DSN is required when DB_TYPE=mysql")
		}
		return store.NewMySQLStore(dsn)
	case "dynamodb":
		region := os.Getenv("AWS_REGION")
		if region == "" {
			region = "us-west-2"
		}
		table := os.Getenv("DYNAMO_TABLE")
		if table == "" {
			table = "shopping_carts"
		}
		return store.NewDynamoStore(region, table)
	default:
		return nil, fmt.Errorf("DB_TYPE env var must be 'mysql' or 'dynamodb'")
	}
}

func parsePositiveInt(c *gin.Context, param string) (int, bool) {
	raw := strings.TrimSpace(c.Param(param))
	val, err := strconv.Atoi(raw)
	if err != nil || val < 1 {
		writeError(c, http.StatusBadRequest, "INVALID_INPUT",
			fmt.Sprintf("Invalid %s", param),
			fmt.Sprintf("%s must be a positive integer", param))
		return 0, false
	}
	return val, true
}

func validateProduct(p Product) string {
	if p.ProductID < 1 {
		return "product_id must be >= 1"
	}
	if strings.TrimSpace(p.SKU) == "" || len(p.SKU) > 100 {
		return "sku is required (max 100 chars)"
	}
	if strings.TrimSpace(p.Manufacturer) == "" || len(p.Manufacturer) > 200 {
		return "manufacturer is required (max 200 chars)"
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
	c.JSON(status, ErrorResponse{Error: code, Message: message, Details: details})
}
