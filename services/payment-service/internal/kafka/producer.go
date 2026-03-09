package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/core-bank-mandiri/payment-service/internal/config"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

// Producer wraps the Kafka writer for producing messages
type Producer struct {
	writer *kafka.Writer
}

// NewProducer creates a new Kafka producer
func NewProducer(cfg config.KafkaConfig) (*Producer, error) {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    cfg.BatchSize,
		BatchTimeout: cfg.BatchTimeout,
		Compression:  getCompression(cfg.Compression),
		Async:        false, // Wait for acks for reliability
	}

	return &Producer{writer: writer}, nil
}

func getCompression(compression string) kafka.Compression {
	switch compression {
	case "gzip":
		return kafka.Gzip
	case "snappy":
		return kafka.Snappy
	case "lz4":
		return kafka.Lz4
	case "zstd":
		return kafka.Zstd
	default:
		return 0 // No compression (kafka.NoCompression)
	}
}

// Close closes the producer
func (p *Producer) Close() error {
	return p.writer.Close()
}

// SendMessage sends a message to a Kafka topic
func (p *Producer) SendMessage(ctx context.Context, topic string, key string, value interface{}) error {
	messageValue, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	msg := kafka.Message{
		Key:   []byte(key),
		Value: messageValue,
		Time:  time.Now(),
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("failed to write message to Kafka: %w", err)
	}

	return nil
}

// Event types for payment service
type TransactionCreatedEvent struct {
	Metadata       EventMetadata `json:"metadata"`
	TransactionID  string        `json:"transaction_id"`
	Reference      string        `json:"reference"`
	IdempotencyKey string        `json:"idempotency_key"`
	TransactionType string       `json:"transaction_type"`
	Amount         Money         `json:"amount"`
	Fee            Money         `json:"fee"`
	Total          Money         `json:"total"`
	FromAccount    AccountRef    `json:"from_account"`
	ToAccount      AccountRef    `json:"to_account"`
	Description    string        `json:"description"`
	Status         string        `json:"status"`
	CreatedAt      time.Time     `json:"created_at"`
}

type TransactionCompletedEvent struct {
	Metadata      EventMetadata `json:"metadata"`
	TransactionID string        `json:"transaction_id"`
	Reference     string        `json:"reference"`
	Amount        Money         `json:"amount"`
	FromAccount   AccountRef    `json:"from_account"`
	ToAccount     AccountRef    `json:"to_account"`
	CompletedAt   time.Time     `json:"completed_at"`
}

type TransactionFailedEvent struct {
	Metadata       EventMetadata `json:"metadata"`
	TransactionID  string        `json:"transaction_id"`
	Reference      string        `json:"reference"`
	FailureCode    string        `json:"failure_code"`
	FailureReason  string        `json:"failure_reason"`
	FailureCategory string       `json:"failure_category"`
	FailedAt       time.Time     `json:"failed_at"`
	CanRetry       bool          `json:"can_retry"`
}

type AccountDebitedEvent struct {
	Metadata     EventMetadata `json:"metadata"`
	AccountID    string        `json:"account_id"`
	AccountNo    string        `json:"account_no"`
	TransactionID string       `json:"transaction_id"`
	Reference    string        `json:"reference"`
	Amount       Money         `json:"amount"`
	BalanceBefore Money        `json:"balance_before"`
	BalanceAfter Money         `json:"balance_after"`
	DebitedAt    time.Time     `json:"debited_at"`
}

type AccountCreditedEvent struct {
	Metadata     EventMetadata `json:"metadata"`
	AccountID    string        `json:"account_id"`
	AccountNo    string        `json:"account_no"`
	TransactionID string       `json:"transaction_id"`
	Reference    string        `json:"reference"`
	Amount       Money         `json:"amount"`
	BalanceBefore Money        `json:"balance_before"`
	BalanceAfter Money         `json:"balance_after"`
	CreditedAt   time.Time     `json:"credited_at"`
}

type NotificationRequestedEvent struct {
	Metadata        EventMetadata    `json:"metadata"`
	NotificationID  string           `json:"notification_id"`
	UserID          string           `json:"user_id"`
	NotificationType string          `json:"notification_type"`
	EventCategory   string           `json:"event_category"`
	Subject         string           `json:"subject"`
	Body            string           `json:"body"`
	TemplateData    map[string]string `json:"template_data"`
	RequestedAt     time.Time        `json:"requested_at"`
}

type EventMetadata struct {
	EventID        string    `json:"event_id"`
	TraceID        string    `json:"trace_id"`
	SpanID         string    `json:"span_id"`
	CorrelationID  string    `json:"correlation_id"`
	OccurredAt     time.Time `json:"occurred_at"`
	Producer       string    `json:"producer"`
	SchemaVersion  string    `json:"schema_version"`
}

type Money struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

type AccountRef struct {
	AccountID   string `json:"account_id"`
	AccountNo   string `json:"account_no"`
	HolderName  string `json:"holder_name"`
}

// PublishTransactionCreated publishes a transaction created event
func (p *Producer) PublishTransactionCreated(ctx context.Context, event TransactionCreatedEvent) error {
	return p.SendMessage(ctx, "transaction.created", event.TransactionID, event)
}

// PublishTransactionCompleted publishes a transaction completed event
func (p *Producer) PublishTransactionCompleted(ctx context.Context, event TransactionCompletedEvent) error {
	return p.SendMessage(ctx, "transaction.completed", event.TransactionID, event)
}

// PublishTransactionFailed publishes a transaction failed event
func (p *Producer) PublishTransactionFailed(ctx context.Context, event TransactionFailedEvent) error {
	return p.SendMessage(ctx, "transaction.failed", event.TransactionID, event)
}

// PublishAccountDebited publishes an account debited event
func (p *Producer) PublishAccountDebited(ctx context.Context, event AccountDebitedEvent) error {
	return p.SendMessage(ctx, "account.debited", event.AccountID, event)
}

// PublishAccountCredited publishes an account credited event
func (p *Producer) PublishAccountCredited(ctx context.Context, event AccountCreditedEvent) error {
	return p.SendMessage(ctx, "account.credited", event.AccountID, event)
}

// PublishNotificationRequested publishes a notification requested event
func (p *Producer) PublishNotificationRequested(ctx context.Context, event NotificationRequestedEvent) error {
	return p.SendMessage(ctx, "notification.request", event.UserID, event)
}

// NewEventMetadata creates a new event metadata with generated IDs
func NewEventMetadata(traceID, producer string) EventMetadata {
	return EventMetadata{
		EventID:       uuid.New().String(),
		TraceID:       traceID,
		SpanID:        uuid.New().String()[:16],
		CorrelationID: uuid.New().String(),
		OccurredAt:    time.Now().UTC(),
		Producer:      producer,
		SchemaVersion: "1.0.0",
	}
}
