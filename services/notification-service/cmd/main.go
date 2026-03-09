package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/segmentio/kafka-go"
)

// Notification Service - Core Bank Mandiri
// Handles all notification delivery (Email, SMS, Push, In-App)
//
// Responsibilities:
// - Consume notification events from Kafka
// - Send email notifications via SMTP
// - Send SMS notifications via SMS gateway
// - Send push notifications via FCM/APNS
// - Store in-app notifications
// - Handle notification preferences

type Config struct {
	KafkaBrokers       []string
	KafkaConsumerGroup string
	PostgresHost       string
	PostgresPort       string
	PostgresDB         string
	PostgresUser       string
	PostgresPassword   string
	SMTPHost           string
	SMTPPort           string
	SMTPUser           string
	SMTPPassword       string
	SMTPFrom           string
	ServerPort         string
}

func loadConfig() *Config {
	return &Config{
		KafkaBrokers:       parseKafkaBrokers(getEnv("KAFKA_BROKERS", "localhost:9092")),
		KafkaConsumerGroup: getEnv("KAFKA_CONSUMER_GROUP", "notification-service-group"),
		PostgresHost:       getEnv("POSTGRES_HOST", "localhost"),
		PostgresPort:       getEnv("POSTGRES_PORT", "5432"),
		PostgresDB:         getEnv("POSTGRES_DB", "core_bank"),
		PostgresUser:       getEnv("POSTGRES_USER", "postgres"),
		PostgresPassword:   getEnv("POSTGRES_PASSWORD", "postgres"),
		SMTPHost:           getEnv("SMTP_HOST", "mailhog"),
		SMTPPort:           getEnv("SMTP_PORT", "1025"),
		SMTPUser:           getEnv("SMTP_USER", ""),
		SMTPPassword:       getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:           getEnv("SMTP_FROM", "noreply@corebank.co.id"),
		ServerPort:         getEnv("SERVER_PORT", "8086"),
	}
}

func parseKafkaBrokers(brokers string) []string {
	if brokers == "" {
		return []string{"localhost:9092"}
	}
	parts := strings.Split(brokers, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return []string{"localhost:9092"}
	}
	return result
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

type NotificationService struct {
	config *Config
	db     *pgxpool.Pool
}

type NotificationEvent struct {
	Metadata         EventMetadata     `json:"metadata"`
	NotificationID   string            `json:"notification_id"`
	UserID           string            `json:"user_id"`
	NotificationType string            `json:"notification_type"`
	EventCategory    string            `json:"event_category"`
	Subject          string            `json:"subject"`
	Body             string            `json:"body"`
	TemplateData     map[string]string `json:"template_data"`
	RequestedAt      time.Time         `json:"requested_at"`
}

type EventMetadata struct {
	EventID       string    `json:"event_id"`
	TraceID       string    `json:"trace_id"`
	SpanID        string    `json:"span_id"`
	CorrelationID string    `json:"correlation_id"`
	OccurredAt    time.Time `json:"occurred_at"`
	Producer      string    `json:"producer"`
	SchemaVersion string    `json:"schema_version"`
}

type Notification struct {
	ID           uuid.UUID
	UserID       string
	Type         string
	Status       string
	Subject      string
	Body         string
	TemplateData map[string]string
	SentAt       time.Time
	DeliveredAt  time.Time
	FailedReason string
	RetryCount   int
	MaxRetries   int
	CreatedAt    time.Time
}

func NewNotificationService(config *Config) (*NotificationService, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Fixed: Remove IPv6 brackets for standard connections
	db, err := pgxpool.New(ctx, fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		config.PostgresUser, config.PostgresPassword,
		config.PostgresHost, config.PostgresPort, config.PostgresDB,
	))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test database connection
	if err := db.Ping(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &NotificationService{
		config: config,
		db:     db,
	}, nil
}

func (ns *NotificationService) Close() {
	if ns.db != nil {
		ns.db.Close()
	}
}

func (ns *NotificationService) Start(ctx context.Context) error {
	log.Println("Starting Notification Service...")

	// Start HTTP server for health checks
	go ns.startHTTPServer()

	// Create Kafka reader with proper configuration
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     ns.config.KafkaBrokers,
		Topic:       "notification.request",
		GroupID:     ns.config.KafkaConsumerGroup,
		MinBytes:    10e3,
		MaxBytes:    10e6,
		StartOffset: kafka.FirstOffset, // Fixed: Read from beginning instead of last
	})
	defer reader.Close()

	log.Printf("Connected to Kafka (brokers: %v, group: %s), waiting for notifications...",
		ns.config.KafkaBrokers, ns.config.KafkaConsumerGroup)

	for {
		select {
		case <-ctx.Done():
			log.Println("Context cancelled, stopping Kafka consumer...")
			return ctx.Err()
		default:
			msg, err := reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				log.Printf("Error fetching message: %v", err)
				time.Sleep(100 * time.Millisecond) // Backoff before retry
				continue
			}

			// Process notification in goroutine
			go func(msg kafka.Message) {
				if err := ns.processNotification(ctx, msg.Value); err != nil {
					log.Printf("Error processing notification: %v", err)
				}
			}(msg)

			// Commit message
			if err := reader.CommitMessages(ctx, msg); err != nil {
				log.Printf("Error committing message: %v", err)
			}
		}
	}
}

func (ns *NotificationService) startHTTPServer() {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "healthy",
			"service": "notification-service",
		})
	})

	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "ready",
			"service": "notification-service",
		})
	})

	server := &http.Server{
		Addr:         ":" + ns.config.ServerPort,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Starting HTTP server on port %s", ns.config.ServerPort)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("HTTP server error: %v", err)
	}
}

func (ns *NotificationService) processNotification(ctx context.Context, data []byte) error {
	var event NotificationEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return fmt.Errorf("failed to unmarshal notification event: %w", err)
	}

	// Validate notification event
	if err := validateNotificationEvent(&event); err != nil {
		return fmt.Errorf("invalid notification event: %w", err)
	}

	log.Printf("Processing notification: %s, Type: %s, User: %s",
		event.NotificationID, event.NotificationType, event.UserID)

	// Check user preferences
	shouldSend, err := ns.checkUserPreferences(ctx, event.UserID, event.NotificationType, event.EventCategory)
	if err != nil {
		log.Printf("Error checking user preferences: %v", err)
		// Continue with default behavior (send notification)
		shouldSend = true
	}

	if !shouldSend {
		log.Printf("User %s has disabled %s notifications for %s",
			event.UserID, event.NotificationType, event.EventCategory)
		return nil
	}

	// Create notification record
	notification := &Notification{
		ID:           uuid.New(),
		UserID:       event.UserID,
		Type:         event.NotificationType,
		Status:       "PENDING",
		Subject:      event.Subject,
		Body:         event.Body,
		TemplateData: event.TemplateData,
		MaxRetries:   3,
		CreatedAt:    time.Now().UTC(),
	}

	if err := ns.saveNotification(ctx, notification); err != nil {
		return fmt.Errorf("failed to save notification: %w", err)
	}

	// Send notification based on type
	var sendErr error
	switch event.NotificationType {
	case "EMAIL":
		sendErr = ns.sendEmail(ctx, notification, event)
	case "SMS":
		sendErr = ns.sendSMS(ctx, notification, event)
	case "PUSH":
		sendErr = ns.sendPush(ctx, notification, event)
	case "IN_APP":
		sendErr = ns.saveInAppNotification(ctx, notification, event)
	default:
		sendErr = fmt.Errorf("unknown notification type: %s", event.NotificationType)
	}

	// Update notification status
	if sendErr != nil {
		notification.Status = "FAILED"
		notification.FailedReason = sendErr.Error()
		notification.RetryCount = 0 // Will be incremented on retry
		log.Printf("Failed to send notification %s: %v", notification.ID, sendErr)

		// Check if we should retry
		if notification.RetryCount < notification.MaxRetries {
			log.Printf("Notification %s will be retried (attempt %d/%d)",
				notification.ID, notification.RetryCount+1, notification.MaxRetries)
		}
	} else {
		notification.Status = "SENT"
		notification.SentAt = time.Now().UTC()
		log.Printf("Notification sent successfully: %s (Type: %s, User: %s)",
			notification.ID, notification.Type, notification.UserID)
	}

	return ns.updateNotificationStatus(ctx, notification)
}

// validateNotificationEvent validates the notification event structure
func validateNotificationEvent(event *NotificationEvent) error {
	if event.UserID == "" {
		return fmt.Errorf("user_id is required")
	}

	if _, err := uuid.Parse(event.UserID); err != nil {
		return fmt.Errorf("invalid user_id format: must be a valid UUID")
	}

	if event.NotificationID == "" {
		return fmt.Errorf("notification_id is required")
	}

	if event.NotificationType == "" {
		return fmt.Errorf("notification_type is required")
	}

	validTypes := map[string]bool{
		"EMAIL":  true,
		"SMS":    true,
		"PUSH":   true,
		"IN_APP": true,
	}

	if !validTypes[event.NotificationType] {
		return fmt.Errorf("invalid notification_type: %s (must be EMAIL, SMS, PUSH, or IN_APP)",
			event.NotificationType)
	}

	if event.Body == "" {
		return fmt.Errorf("body is required")
	}

	if event.EventCategory == "" {
		return fmt.Errorf("event_category is required")
	}

	return nil
}

func (ns *NotificationService) checkUserPreferences(ctx context.Context, userID, notificationType, eventCategory string) (bool, error) {
	query := `
		SELECT enabled FROM notification_preferences
		WHERE user_id = $1 AND notification_type = $2 AND event_category = $3
	`

	var enabled bool
	err := ns.db.QueryRow(ctx, query, userID, notificationType, eventCategory).Scan(&enabled)
	if err != nil {
		// Default to enabled if no preference found or user doesn't exist
		if err == pgx.ErrNoRows {
			return true, nil
		}
		return true, fmt.Errorf("error checking preferences: %w", err)
	}

	return enabled, nil
}

func (ns *NotificationService) saveNotification(ctx context.Context, notification *Notification) error {
	query := `
		INSERT INTO notifications (
			id, user_id, notification_type, status, subject, body,
			template_data, created_at, retry_count, max_retries
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := ns.db.Exec(ctx, query,
		notification.ID, notification.UserID, notification.Type,
		notification.Status, notification.Subject, notification.Body,
		notification.TemplateData, notification.CreatedAt,
		notification.RetryCount, notification.MaxRetries,
	)

	return err
}

func (ns *NotificationService) updateNotificationStatus(ctx context.Context, notification *Notification) error {
	query := `
		UPDATE notifications
		SET status = $1, sent_at = $2, delivered_at = $3, failed_reason = $4, retry_count = $5
		WHERE id = $6
	`

	_, err := ns.db.Exec(ctx, query,
		notification.Status, notification.SentAt, notification.DeliveredAt,
		notification.FailedReason, notification.RetryCount, notification.ID,
	)

	return err
}

func (ns *NotificationService) sendEmail(ctx context.Context, notification *Notification, event NotificationEvent) error {
	// In production, integrate with SendGrid, SES, or SMTP server
	// For now, just log the email
	log.Printf("📧 Sending EMAIL to user %s: Subject=%s, Body=%s",
		notification.UserID, notification.Subject, notification.Body)
	log.Printf("   SMTP Config: Host=%s, Port=%s, From=%s",
		ns.config.SMTPHost, ns.config.SMTPPort, ns.config.SMTPFrom)

	// Simulate email sending delay
	time.Sleep(100 * time.Millisecond)

	return nil
}

func (ns *NotificationService) sendSMS(ctx context.Context, notification *Notification, event NotificationEvent) error {
	// In production, integrate with Twilio, Vonage, or local SMS provider
	log.Printf("📱 Sending SMS to user %s: %s", notification.UserID, notification.Body)

	time.Sleep(50 * time.Millisecond)
	return nil
}

func (ns *NotificationService) sendPush(ctx context.Context, notification *Notification, event NotificationEvent) error {
	// In production, integrate with FCM (Firebase Cloud Messaging) or APNS
	log.Printf("🔔 Sending PUSH notification to user %s: %s", notification.UserID, notification.Subject)

	time.Sleep(50 * time.Millisecond)
	return nil
}

func (ns *NotificationService) saveInAppNotification(ctx context.Context, notification *Notification, event NotificationEvent) error {
	// In-app notifications are already saved in the notifications table
	// Could also cache in Redis for fast retrieval
	log.Printf("💬 In-app notification saved for user %s: %s", notification.UserID, notification.Subject)

	return nil
}

func main() {
	config := loadConfig()

	service, err := NewNotificationService(config)
	if err != nil {
		log.Fatalf("Failed to create notification service: %v", err)
	}
	defer service.Close()

	// Create context with cancellation for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("Shutdown signal received, stopping service...")
		cancel()

		// Give some time for cleanup
		time.Sleep(2 * time.Second)
	}()

	log.Println("Notification Service starting up...")
	if err := service.Start(ctx); err != nil {
		if err == context.Canceled {
			log.Println("Notification Service stopped gracefully")
		} else {
			log.Fatalf("Notification service failed: %v", err)
		}
	}
}
