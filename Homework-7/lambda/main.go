package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

const paymentDelay = 3 * time.Second

type Order struct {
	OrderID    string `json:"order_id"`
	CustomerID string `json:"customer_id"`
	CreatedAt  string `json:"created_at"`
}

func handler(ctx context.Context, snsEvent events.SNSEvent) error {
	for _, record := range snsEvent.Records {
		var order Order
		if err := json.Unmarshal([]byte(record.SNS.Message), &order); err != nil {
			log.Printf("failed to parse order: %v", err)
			continue
		}

		log.Printf("processing order %s for customer %s", order.OrderID, order.CustomerID)

		// Simulate 3-second payment verification
		time.Sleep(paymentDelay)

		log.Printf("completed order %s", order.OrderID)
	}
	return nil
}

func main() {
	lambda.Start(handler)
}
