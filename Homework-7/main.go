package main

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/gin-gonic/gin"
)

const paymentDelay = 3 * time.Second

type OrderItem struct {
	SKU      string `json:"sku" binding:"required"`
	Quantity int    `json:"quantity" binding:"required,min=1"`
}

type OrderRequest struct {
	CustomerID string      `json:"customer_id" binding:"required"`
	Items      []OrderItem `json:"items" binding:"required,min=1,dive"`
}

type OrderResponse struct {
	OrderID string `json:"order_id"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type Server struct {
	paymentSlots chan struct{}
	snsClient    *sns.Client
	snsTopicARN  string
}

func main() {
	rand.Seed(time.Now().UnixNano())

	appMode := getEnv("APP_MODE", "receiver")
	if appMode == "processor" {
		runProcessor()
		return
	}

	server := &Server{
		paymentSlots: make(chan struct{}, getEnvAsInt("SYNC_PAYMENT_CONCURRENCY", 15)),
		snsTopicARN:  os.Getenv("SNS_TOPIC_ARN"),
	}

	if server.snsTopicARN != "" {
		client, err := newSNSClient(context.Background())
		if err != nil {
			log.Printf("failed to create SNS client, async orders will be logged locally: %v", err)
		} else {
			server.snsClient = client
		}
	}

	router := gin.Default()
	router.GET("/health", server.handleHealth)
	router.POST("/orders/sync", server.handleSyncOrder)
	router.POST("/orders/async", server.handleAsyncOrder)

	port := getEnv("PORT", "8080")
	log.Printf("starting receiver on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}

func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) handleSyncOrder(c *gin.Context) {
	var req OrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	orderID := generateOrderID()
	log.Printf("received sync order %s for customer %s", orderID, req.CustomerID)

	s.paymentSlots <- struct{}{}
	defer func() {
		<-s.paymentSlots
	}()

	time.Sleep(paymentDelay)

	c.JSON(http.StatusOK, OrderResponse{
		OrderID: orderID,
		Status:  "completed",
		Message: "payment verified and order completed",
	})
}

func (s *Server) handleAsyncOrder(c *gin.Context) {
	var req OrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	orderID := generateOrderID()
	messageBody, err := json.Marshal(gin.H{
		"order_id":    orderID,
		"customer_id": req.CustomerID,
		"items":       req.Items,
		"created_at":  time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encode order"})
		return
	}

	if s.snsClient == nil || s.snsTopicARN == "" {
		log.Printf("SNS not configured, accepting async order locally: %s %s", orderID, string(messageBody))
		c.JSON(http.StatusAccepted, OrderResponse{
			OrderID: orderID,
			Status:  "accepted",
		})
		return
	}

	_, err = s.snsClient.Publish(c.Request.Context(), &sns.PublishInput{
		TopicArn: &s.snsTopicARN,
		Message:  stringPtr(string(messageBody)),
		MessageAttributes: map[string]types.MessageAttributeValue{
			"order_id": {
				DataType:    stringPtr("String"),
				StringValue: stringPtr(orderID),
			},
		},
	})
	if err != nil {
		log.Printf("failed to publish order %s to SNS: %v", orderID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to publish order"})
		return
	}

	c.JSON(http.StatusAccepted, OrderResponse{
		OrderID: orderID,
		Status:  "accepted",
	})
}

func newSNSClient(ctx context.Context) (*sns.Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(getEnv("AWS_REGION", "us-west-2")))
	if err != nil {
		return nil, err
	}

	return sns.NewFromConfig(cfg), nil
}

func generateOrderID() string {
	return "order-" + strconv.FormatInt(time.Now().UnixNano(), 36) + strconv.Itoa(rand.Intn(1000))
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getEnvAsInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}

	return parsed
}

func stringPtr(value string) *string {
	return &value
}
