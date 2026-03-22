package store

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type MySQLStore struct {
	db *sql.DB
}

func NewMySQLStore(dsn string) (*MySQLStore, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	// pool config - tuned for db.t3.micro
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, err
	}

	if err := createTables(db); err != nil {
		return nil, err
	}

	return &MySQLStore{db: db}, nil
}

func createTables(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS shopping_carts (
			id INT AUTO_INCREMENT PRIMARY KEY,
			customer_id INT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_customer_id (customer_id)
		)`,
		`CREATE TABLE IF NOT EXISTS cart_items (
			id INT AUTO_INCREMENT PRIMARY KEY,
			cart_id INT NOT NULL,
			product_id INT NOT NULL,
			quantity INT NOT NULL DEFAULT 1,
			added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (cart_id) REFERENCES shopping_carts(id) ON DELETE CASCADE,
			UNIQUE KEY uk_cart_product (cart_id, product_id)
		)`,
	}
	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

func (s *MySQLStore) CreateCart(ctx context.Context, customerID int) (int, error) {
	res, err := s.db.ExecContext(ctx, "INSERT INTO shopping_carts (customer_id) VALUES (?)", customerID)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

func (s *MySQLStore) GetCart(ctx context.Context, cartID int) (*ShoppingCart, error) {
	// single query with LEFT JOIN so we get cart even if it has no items
	rows, err := s.db.QueryContext(ctx, `
		SELECT sc.id, sc.customer_id, sc.created_at, sc.updated_at,
		       ci.product_id, ci.quantity
		FROM shopping_carts sc
		LEFT JOIN cart_items ci ON sc.id = ci.cart_id
		WHERE sc.id = ?`, cartID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cart *ShoppingCart
	for rows.Next() {
		var (
			id         int
			customerID int
			createdAt  time.Time
			updatedAt  time.Time
			productID  sql.NullInt64
			quantity   sql.NullInt64
		)
		if err := rows.Scan(&id, &customerID, &createdAt, &updatedAt, &productID, &quantity); err != nil {
			return nil, err
		}
		if cart == nil {
			cart = &ShoppingCart{
				ShoppingCartID: id,
				CustomerID:     customerID,
				Items:          []CartItem{},
				CreatedAt:      createdAt,
				UpdatedAt:      updatedAt,
			}
		}
		if productID.Valid {
			cart.Items = append(cart.Items, CartItem{
				ProductID: int(productID.Int64),
				Quantity:  int(quantity.Int64),
			})
		}
	}

	if cart == nil {
		return nil, ErrCartNotFound
	}
	return cart, nil
}

func (s *MySQLStore) AddItem(ctx context.Context, cartID int, productID int, quantity int) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// lock the cart row so concurrent adds don't conflict
	var id int
	err = tx.QueryRowContext(ctx, "SELECT id FROM shopping_carts WHERE id = ? FOR UPDATE", cartID).Scan(&id)
	if err == sql.ErrNoRows {
		return ErrCartNotFound
	}
	if err != nil {
		return err
	}

	// upsert - if same product already in cart, just add to quantity
	_, err = tx.ExecContext(ctx,
		`INSERT INTO cart_items (cart_id, product_id, quantity) VALUES (?, ?, ?)
		 ON DUPLICATE KEY UPDATE quantity = quantity + VALUES(quantity)`,
		cartID, productID, quantity)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, "UPDATE shopping_carts SET updated_at = NOW() WHERE id = ?", cartID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *MySQLStore) Close() error {
	return s.db.Close()
}
