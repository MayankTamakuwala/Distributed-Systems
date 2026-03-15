package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

func runProcessor() {
	queueURL := os.Getenv("SQS_QUEUE_URL")
	if queueURL == "" {
		log.Fatal("SQS_QUEUE_URL is required when APP_MODE=processor")
	}

	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(getEnv("AWS_REGION", "us-west-2")))
	if err != nil {
		log.Fatalf("failed to load AWS config: %v", err)
	}

	client := sqs.NewFromConfig(cfg)
	workerCount := getEnvAsInt("WORKER_COUNT", 5)
	jobs := make(chan sqsTypesMessage, workerCount*2)

	for i := 1; i <= workerCount; i++ {
		go workerLoop(i, client, queueURL, jobs)
	}

	log.Printf("starting processor with %d workers", workerCount)

	for {
		output, err := client.ReceiveMessage(context.Background(), &sqs.ReceiveMessageInput{
			QueueUrl:            &queueURL,
			MaxNumberOfMessages: 10,
			WaitTimeSeconds:     20,
			VisibilityTimeout:   30,
		})
		if err != nil {
			log.Printf("failed to receive messages: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}

		if len(output.Messages) == 0 {
			continue
		}

		for _, msg := range output.Messages {
			jobs <- sqsTypesMessage{Body: msg.Body, ReceiptHandle: msg.ReceiptHandle}
		}
	}
}

type sqsTypesMessage struct {
	Body          *string
	ReceiptHandle *string
}

func workerLoop(id int, client *sqs.Client, queueURL string, jobs <-chan sqsTypesMessage) {
	for msg := range jobs {
		if msg.Body == nil || msg.ReceiptHandle == nil {
			log.Printf("worker %d received invalid message", id)
			continue
		}

		log.Printf("worker %d processing message: %s", id, *msg.Body)
		time.Sleep(paymentDelay)

		_, err := client.DeleteMessage(context.Background(), &sqs.DeleteMessageInput{
			QueueUrl:      &queueURL,
			ReceiptHandle: msg.ReceiptHandle,
		})
		if err != nil {
			log.Printf("worker %d failed to delete message: %v", id, err)
			continue
		}

		log.Printf("worker %d completed message", id)
	}
}
