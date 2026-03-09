package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/redis/go-redis/v9"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Fraud Detection Service - Core Bank Mandiri
// Real-time fraud detection using stream processing
//
// Responsibilities:
// - Analyze transaction events in real-time
// - Detect suspicious activity patterns
// - Calculate risk scores
// - Generate fraud alerts
// - Block high-risk transactions

type Config struct {
	KafkaBrokers    []string
	RedisHost       string
	RedisPort       string
	PostgresHost    string
	PostgresPort    string
	PostgresDB      string
	PostgresUser    string
	PostgresPassword string
	ConsumerGroup   string
}

func loadConfig() *Config {
	return &Config{
		KafkaBrokers:    []string{getEnv("KAFKA_BROKERS", "localhost:9092")},
		RedisHost:       getEnv("REDIS_HOST", "localhost"),
		RedisPort:       getEnv("REDIS_PORT", "6379"),
		PostgresHost:    getEnv("POSTGRES_HOST", "localhost"),
		PostgresPort:    getEnv("POSTGRES_PORT", "5432"),
		PostgresDB:      getEnv("POSTGRES_DB", "core_bank"),
		PostgresUser:    getEnv("POSTGRES_USER", "postgres"),
		PostgresPassword: getEnv("POSTGRES_PASSWORD", "postgres"),
		ConsumerGroup:   getEnv("CONSUMER_GROUP", "fraud-detection-group"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

type FraudDetector struct {
	config      *Config
	redis       *redis.Client
	db          *pgxpool.Pool
	riskRules   []RiskRule
	mu          sync.RWMutex
}

type RiskRule struct {
	ID          string
	Name        string
	Description string
	Weight      int
	Check       func(TransactionEvent) bool
}

type TransactionEvent struct {
	Metadata       EventMetadata `json:"metadata"`
	TransactionID  string        `json:"transaction_id"`
	Reference      string        `json:"reference"`
	TransactionType string       `json:"transaction_type"`
	Amount         Money         `json:"amount"`
	FromAccount    AccountRef    `json:"from_account"`
	ToAccount      AccountRef    `json:"to_account"`
	CreatedAt      time.Time     `json:"created_at"`
}

type EventMetadata struct {
	EventID       string    `json:"event_id"`
	TraceID       string    `json:"trace_id"`
	OccurredAt    time.Time `json:"occurred_at"`
	Producer      string    `json:"producer"`
}

type Money struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

type AccountRef struct {
	AccountID  string `json:"account_id"`
	AccountNo  string `json:"account_no"`
	HolderName string `json:"holder_name"`
}

type FraudAlert struct {
	ID            uuid.UUID
	TransactionID string
	UserID        string
	AlertType     string
	RiskScore     int
	Status        string
	Reason        string
	RulesTriggered []string
	CreatedAt     time.Time
}

func NewFraudDetector(config *Config) (*FraudDetector, error) {
	// Initialize Redis
	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", config.RedisHost, config.RedisPort),
	})

	// Initialize database
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := pgxpool.New(ctx, fmt.Sprintf(
		"postgres://%s:%s@[%s]:%s/%s?sslmode=disable",
		config.PostgresUser, config.PostgresPassword,
		config.PostgresHost, config.PostgresPort, config.PostgresDB,
	))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	fd := &FraudDetector{
		config:    config,
		redis:     rdb,
		db:        db,
		riskRules: initRiskRules(),
	}

	return fd, nil
}

func initRiskRules() []RiskRule {
	return []RiskRule{
		{
			ID:          "HIGH_AMOUNT",
			Name:        "High Amount Transaction",
			Description: "Transaction amount exceeds threshold",
			Weight:      30,
			Check: func(tx TransactionEvent) bool {
				amount := parseAmount(tx.Amount.Amount)
				return amount > 50000000 // 50 million IDR
			},
		},
		{
			ID:          "VELOCITY_CHECK",
			Name:        "High Velocity",
			Description: "Multiple transactions in short time",
			Weight:      25,
			Check: func(tx TransactionEvent) bool {
				// This would check Redis for transaction count in last hour
				return false // Placeholder
			},
		},
		{
			ID:          "UNUSUAL_TIME",
			Name:        "Unusual Time",
			Description: "Transaction at unusual hours",
			Weight:      15,
			Check: func(tx TransactionEvent) bool {
				hour := tx.CreatedAt.Hour()
				return hour >= 1 && hour <= 5 // 1 AM to 5 AM
			},
		},
		{
			ID:          "ROUND_AMOUNT",
			Name:        "Round Amount",
			Description: "Suspiciously round transaction amount",
			Weight:      10,
			Check: func(tx TransactionEvent) bool {
				amount := parseAmount(tx.Amount.Amount)
				return amount >= 10000000 && amount%10000000 == 0
			},
		},
		{
			ID:          "NEW_BENEFICIARY",
			Name:        "New Beneficiary",
			Description: "First transaction to this beneficiary",
			Weight:      20,
			Check: func(tx TransactionEvent) bool {
				// Check if this is first transaction to this account
				return false // Placeholder
			},
		},
	}
}

func parseAmount(amountStr string) int64 {
	var amount int64
	fmt.Sscanf(amountStr, "%d", &amount)
	return amount
}

func (fd *FraudDetector) Start() error {
	log.Println("Starting Fraud Detection Service...")

	// Create Kafka reader
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   fd.config.KafkaBrokers,
		Topic:     "transaction.created",
		GroupID:   fd.config.ConsumerGroup,
		MinBytes:  10e3,
		MaxBytes:  10e6,
		StartOffset: kafka.LastOffset,
	})
	defer reader.Close()

	log.Println("Connected to Kafka, waiting for transactions...")

	ctx := context.Background()

	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			log.Printf("Error fetching message: %v", err)
			continue
		}

		// Process transaction in goroutine for concurrency
		go func(msg kafka.Message) {
			if err := fd.processTransaction(ctx, msg.Value); err != nil {
				log.Printf("Error processing transaction: %v", err)
			}
		}(msg)

		// Commit message
		if err := reader.CommitMessages(ctx, msg); err != nil {
			log.Printf("Error committing message: %v", err)
		}
	}
}

func (fd *FraudDetector) processTransaction(ctx context.Context, data []byte) error {
	var tx TransactionEvent
	if err := json.Unmarshal(data, &tx); err != nil {
		return fmt.Errorf("failed to unmarshal transaction: %w", err)
	}

	log.Printf("Analyzing transaction: %s, Amount: %s", tx.TransactionID, tx.Amount.Amount)

	// Calculate risk score
	riskScore, rulesTriggered := fd.calculateRiskScore(tx)

	log.Printf("Transaction %s risk score: %d, Rules triggered: %v", tx.TransactionID, riskScore, rulesTriggered)

	// Update user risk profile in Redis
	fd.updateUserRiskProfile(tx.FromAccount.AccountID, riskScore)

	// Generate alert if high risk
	if riskScore >= 50 {
		alert := fd.createFraudAlert(tx, riskScore, rulesTriggered)
		if err := fd.saveAlert(ctx, alert); err != nil {
			return fmt.Errorf("failed to save alert: %w", err)
		}

		// Publish fraud alert event
		if err := fd.publishFraudAlert(ctx, alert); err != nil {
			return fmt.Errorf("failed to publish alert: %w", err)
		}

		// Block transaction if very high risk
		if riskScore >= 80 {
			if err := fd.blockTransaction(ctx, tx.TransactionID); err != nil {
				log.Printf("Error blocking transaction: %v", err)
			}
		}
	}

	return nil
}

func (fd *FraudDetector) calculateRiskScore(tx TransactionEvent) (int, []string) {
	totalScore := 0
	var rulesTriggered []string

	for _, rule := range fd.riskRules {
		if rule.Check(tx) {
			totalScore += rule.Weight
			rulesTriggered = append(rulesTriggered, rule.ID)
		}
	}

	// Add historical risk factor
	historicalRisk := fd.getHistoricalRisk(tx.FromAccount.AccountID)
	totalScore += historicalRisk

	// Cap at 100
	if totalScore > 100 {
		totalScore = 100
	}

	return totalScore, rulesTriggered
}

func (fd *FraudDetector) getHistoricalRisk(accountID string) int {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	key := fmt.Sprintf("risk:history:%s", accountID)
	score, err := fd.redis.Get(ctx, key).Int()
	if err != nil {
		return 0
	}
	return score / 10 // Scale down historical risk
}

func (fd *FraudDetector) updateUserRiskProfile(accountID string, riskScore int) {
	key := fmt.Sprintf("risk:profile:%s", accountID)
	
	// Use exponential moving average
	existingScore, _ := fd.redis.Get(context.Background(), key).Int()
	if existingScore == 0 {
		existingScore = riskScore
	} else {
		// EMA with alpha = 0.3
		existingScore = int(float64(existingScore)*0.7 + float64(riskScore)*0.3)
	}

	fd.redis.Set(context.Background(), key, existingScore, 24*time.Hour)

	// Also update historical risk
	historyKey := fmt.Sprintf("risk:history:%s", accountID)
	fd.redis.IncrBy(context.Background(), historyKey, int64(riskScore))
	fd.redis.Expire(context.Background(), historyKey, 7*24*time.Hour)
}

func (fd *FraudDetector) createFraudAlert(tx TransactionEvent, riskScore int, rulesTriggered []string) *FraudAlert {
	alertType := "VELOCITY"
	if len(rulesTriggered) > 0 {
		alertType = rulesTriggered[0]
	}

	reason := fmt.Sprintf("Risk score %d based on rules: %v", riskScore, rulesTriggered)

	return &FraudAlert{
		ID:            uuid.New(),
		TransactionID: tx.TransactionID,
		UserID:        tx.FromAccount.AccountID,
		AlertType:     alertType,
		RiskScore:     riskScore,
		Status:        "PENDING",
		Reason:        reason,
		RulesTriggered: rulesTriggered,
		CreatedAt:     time.Now().UTC(),
	}
}

func (fd *FraudDetector) saveAlert(ctx context.Context, alert *FraudAlert) error {
	query := `
		INSERT INTO fraud_alerts (
			id, transaction_id, user_id, alert_type, risk_score,
			status, reason, rules_triggered, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := fd.db.Exec(ctx, query,
		alert.ID, alert.TransactionID, alert.UserID, alert.AlertType,
		alert.RiskScore, alert.Status, alert.Reason, alert.RulesTriggered, alert.CreatedAt,
	)

	return err
}

func (fd *FraudDetector) publishFraudAlert(ctx context.Context, alert *FraudAlert) error {
	// Create Kafka writer
	writer := &kafka.Writer{
		Addr:     kafka.TCP(fd.config.KafkaBrokers...),
		Topic:    "fraud.alert",
		Balancer: &kafka.LeastBytes{},
	}
	defer writer.Close()

	event := map[string]interface{}{
		"metadata": map[string]interface{}{
			"event_id":      uuid.New().String(),
			"trace_id":      "",
			"occurred_at":   time.Now().UTC(),
			"producer":      "fraud-detection-service",
			"schema_version": "1.0.0",
		},
		"alert_id":        alert.ID.String(),
		"transaction_id":  alert.TransactionID,
		"user_id":         alert.UserID,
		"risk_score":      alert.RiskScore,
		"alert_type":      alert.AlertType,
		"rules_triggered": alert.RulesTriggered,
		"recommended_action": getRecommendedAction(alert.RiskScore),
		"triggered_at":    alert.CreatedAt,
	}

	data, _ := json.Marshal(event)

	return writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(alert.TransactionID),
		Value: data,
		Time:  time.Now(),
	})
}

func getRecommendedAction(riskScore int) string {
	if riskScore >= 80 {
		return "BLOCK"
	} else if riskScore >= 50 {
		return "REVIEW"
	}
	return "ALLOW"
}

func (fd *FraudDetector) blockTransaction(ctx context.Context, transactionID string) error {
	// Publish to transaction block topic
	writer := &kafka.Writer{
		Addr:  kafka.TCP(fd.config.KafkaBrokers...),
		Topic: "transaction.block",
	}
	defer writer.Close()

	event := map[string]interface{}{
		"transaction_id": transactionID,
		"blocked_at":     time.Now().UTC(),
		"reason":         "High fraud risk score",
	}

	data, _ := json.Marshal(event)

	return writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(transactionID),
		Value: data,
		Time:  time.Now(),
	})
}

func main() {
	config := loadConfig()

	detector, err := NewFraudDetector(config)
	if err != nil {
		log.Fatalf("Failed to create fraud detector: %v", err)
	}

	// Handle graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("Shutting down...")
	}()

	if err := detector.Start(); err != nil {
		log.Fatalf("Fraud detector failed: %v", err)
	}
}
