package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/google/uuid"
)

// Simple Kafka Producer for testing notification-service

type EventMetadata struct {
	EventID       string    `json:"event_id"`
	TraceID       string    `json:"trace_id"`
	SpanID        string    `json:"span_id"`
	CorrelationID string    `json:"correlation_id"`
	OccurredAt    time.Time `json:"occurred_at"`
	Producer      string    `json:"producer"`
	SchemaVersion string    `json:"schema_version"`
}

type NotificationEvent struct {
	Metadata       EventMetadata     `json:"metadata"`
	NotificationID string            `json:"notification_id"`
	UserID         string            `json:"user_id"`
	NotificationType string          `json:"notification_type"`
	EventCategory  string            `json:"event_category"`
	Subject        string            `json:"subject"`
	Body           string            `json:"body"`
	TemplateData   map[string]string `json:"template_data"`
	RequestedAt    time.Time         `json:"requested_at"`
}

func main() {
	// Kafka configuration
	brokers := []string{"localhost:9092"}
	topic := "notification.request"

	// Create Kafka writer
	writer := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}
	defer writer.Close()

	// Create test notification
	event := NotificationEvent{
		Metadata: EventMetadata{
			EventID:       uuid.New().String(),
			TraceID:       uuid.New().String(),
			SpanID:        uuid.New().String()[:16],
			CorrelationID: uuid.New().String(),
			OccurredAt:    time.Now().UTC(),
			Producer:      "test-producer",
			SchemaVersion: "1.0.0",
		},
		NotificationID:   uuid.New().String(),
		UserID:           uuid.New().String(),
		NotificationType: "IN_APP",
		EventCategory:    "TRANSACTION",
		Subject:          "Test Notification from Go Producer",
		Body:             "This is a test notification sent via Kafka using Go producer!",
		TemplateData: map[string]string{
			"amount":   "1000000",
			"currency": "IDR",
		},
		RequestedAt: time.Now().UTC(),
	}

	// Marshal to JSON
	data, err := json.Marshal(event)
	if err != nil {
		log.Fatalf("Failed to marshal event: %v", err)
	}

	// Send message
	ctx := context.Background()
	err = writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(event.UserID),
		Value: data,
		Time:  time.Now(),
	})

	if err != nil {
		log.Fatalf("Failed to send message: %v", err)
	}

	fmt.Println("✅ Notification sent successfully!")
	fmt.Printf("   Topic: %s\n", topic)
	fmt.Printf("   Notification ID: %s\n", event.NotificationID)
	fmt.Printf("   User ID: %s\n", event.UserID)
	fmt.Printf("   Type: %s\n", event.NotificationType)
	fmt.Printf("   Subject: %s\n", event.Subject)
	fmt.Printf("   Body: %s\n", event.Body)
}
