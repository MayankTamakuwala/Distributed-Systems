package store

import (
	"context"
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DynamoStore struct {
	client    *dynamodb.Client
	tableName string
	nextID    atomic.Int64 // simple counter for cart IDs (works for single instance)
}

func NewDynamoStore(region, tableName string) (*DynamoStore, error) {
	cfg, err := awsconfig.LoadDefaultConfig(context.Background(), awsconfig.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load aws config: %w", err)
	}
	return &DynamoStore{
		client:    dynamodb.NewFromConfig(cfg),
		tableName: tableName,
	}, nil
}

func (s *DynamoStore) CreateCart(ctx context.Context, customerID int) (int, error) {
	id := int(s.nextID.Add(1))
	now := time.Now().UTC().Format(time.RFC3339)

	_, err := s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &s.tableName,
		Item: map[string]types.AttributeValue{
			"cart_id":     &types.AttributeValueMemberS{Value: strconv.Itoa(id)},
			"customer_id": &types.AttributeValueMemberN{Value: strconv.Itoa(customerID)},
			"items":       &types.AttributeValueMemberL{Value: []types.AttributeValue{}},
			"created_at":  &types.AttributeValueMemberS{Value: now},
			"updated_at":  &types.AttributeValueMemberS{Value: now},
		},
	})
	if err != nil {
		return 0, fmt.Errorf("PutItem failed: %w", err)
	}
	return id, nil
}

func (s *DynamoStore) GetCart(ctx context.Context, cartID int) (*ShoppingCart, error) {
	out, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &s.tableName,
		Key: map[string]types.AttributeValue{
			"cart_id": &types.AttributeValueMemberS{Value: strconv.Itoa(cartID)},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("GetItem failed: %w", err)
	}
	if out.Item == nil {
		return nil, ErrCartNotFound
	}

	// manually unmarshal since the DynamoDB attribute value types are verbose
	cart := &ShoppingCart{
		ShoppingCartID: cartID,
		Items:          []CartItem{},
	}
	if v, ok := out.Item["customer_id"].(*types.AttributeValueMemberN); ok {
		cart.CustomerID, _ = strconv.Atoi(v.Value)
	}
	if v, ok := out.Item["created_at"].(*types.AttributeValueMemberS); ok {
		cart.CreatedAt, _ = time.Parse(time.RFC3339, v.Value)
	}
	if v, ok := out.Item["updated_at"].(*types.AttributeValueMemberS); ok {
		cart.UpdatedAt, _ = time.Parse(time.RFC3339, v.Value)
	}
	if v, ok := out.Item["items"].(*types.AttributeValueMemberL); ok {
		for _, elem := range v.Value {
			if m, ok := elem.(*types.AttributeValueMemberM); ok {
				item := CartItem{}
				if pid, ok := m.Value["product_id"].(*types.AttributeValueMemberN); ok {
					item.ProductID, _ = strconv.Atoi(pid.Value)
				}
				if qty, ok := m.Value["quantity"].(*types.AttributeValueMemberN); ok {
					item.Quantity, _ = strconv.Atoi(qty.Value)
				}
				cart.Items = append(cart.Items, item)
			}
		}
	}
	return cart, nil
}

func (s *DynamoStore) AddItem(ctx context.Context, cartID int, productID int, quantity int) error {
	// need to fetch first, modify items list, then write back
	// not the most efficient but keeps it simple
	cart, err := s.GetCart(ctx, cartID)
	if err != nil {
		return err
	}

	// check if product already in cart
	found := false
	for i, item := range cart.Items {
		if item.ProductID == productID {
			cart.Items[i].Quantity += quantity
			found = true
			break
		}
	}
	if !found {
		cart.Items = append(cart.Items, CartItem{ProductID: productID, Quantity: quantity})
	}

	// rebuild the items attribute value list
	itemsAV := make([]types.AttributeValue, len(cart.Items))
	for i, item := range cart.Items {
		itemsAV[i] = &types.AttributeValueMemberM{
			Value: map[string]types.AttributeValue{
				"product_id": &types.AttributeValueMemberN{Value: strconv.Itoa(item.ProductID)},
				"quantity":   &types.AttributeValueMemberN{Value: strconv.Itoa(item.Quantity)},
			},
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)

	// NOTE: "items" is a reserved word in DynamoDB so we have to use
	// ExpressionAttributeNames to alias it. Took a while to figure this out.
	_, err = s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: &s.tableName,
		Key: map[string]types.AttributeValue{
			"cart_id": &types.AttributeValueMemberS{Value: strconv.Itoa(cartID)},
		},
		UpdateExpression: aws.String("SET #items = :items, updated_at = :now"),
		ExpressionAttributeNames: map[string]string{
			"#items": "items",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":items": &types.AttributeValueMemberL{Value: itemsAV},
			":now":   &types.AttributeValueMemberS{Value: now},
		},
	})
	if err != nil {
		return fmt.Errorf("UpdateItem failed: %w", err)
	}
	return nil
}

func (s *DynamoStore) Close() error {
	return nil
}
