package store

import (
	"context"
	"errors"
	"time"
)

var ErrCartNotFound = errors.New("cart not found")

type CartItem struct {
	ProductID int `json:"product_id"`
	Quantity  int `json:"quantity"`
}

type ShoppingCart struct {
	ShoppingCartID int        `json:"shopping_cart_id"`
	CustomerID     int        `json:"customer_id"`
	Items          []CartItem `json:"items"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// CartStore is implemented by both MySQLStore and DynamoStore
// so we can swap backends via DB_TYPE env var
type CartStore interface {
	CreateCart(ctx context.Context, customerID int) (int, error)
	GetCart(ctx context.Context, cartID int) (*ShoppingCart, error)
	AddItem(ctx context.Context, cartID int, productID int, quantity int) error
	Close() error
}
