package service

import (
	"context"
	"fmt"
	"time"

	"github.com/core-bank-mandiri/payment-service/internal/config"
	"github.com/core-bank-mandiri/payment-service/internal/kafka"
	"github.com/core-bank-mandiri/payment-service/internal/repository"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// PaymentService handles payment processing logic
type PaymentService struct {
	db       *repository.PostgresRepository
	rdb      *redis.Client
	producer *kafka.Producer
	cfg      *config.Config
}

// NewPaymentService creates a new payment service
func NewPaymentService(db *repository.PostgresRepository, rdb *redis.Client, producer *kafka.Producer, cfg *config.Config) *PaymentService {
	return &PaymentService{
		db:       db,
		rdb:      rdb,
		producer: producer,
		cfg:      cfg,
	}
}

// TransferRequest contains the request for a transfer
type TransferRequest struct {
	FromAccountID   uuid.UUID
	ToAccountID     uuid.UUID
	ToAccountNumber string
	ToBankCode      string
	ToAccountName   string
	Amount          int64
	Description     string
	IdempotencyKey  string
	TraceID         string
}

// TransferResponse contains the response for a transfer
type TransferResponse struct {
	TransactionID uuid.UUID
	Reference     string
	Status        string
	Amount        int64
	Fee           int64
	Total         int64
}

// InternalTransfer processes an internal transfer (same bank)
func (s *PaymentService) InternalTransfer(ctx context.Context, req TransferRequest) (*TransferResponse, error) {
	// Check idempotency
	if req.IdempotencyKey != "" {
		existingTx, err := s.checkIdempotency(ctx, req.IdempotencyKey)
		if err != nil {
			return nil, fmt.Errorf("failed to check idempotency: %w", err)
		}
		if existingTx != nil {
			return existingTx, nil
		}
	}

	// Validate accounts are the same bank (internal)
	if req.ToBankCode != "" && req.ToBankCode != "MANDIRI" {
		return nil, fmt.Errorf("external transfers must use external transfer endpoint")
	}

	// Get from account
	fromAccount, err := s.db.GetAccountByID(ctx, req.FromAccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get from account: %w", err)
	}
	if fromAccount == nil {
		return nil, fmt.Errorf("from account not found")
	}
	if fromAccount.Status != "ACTIVE" {
		return nil, fmt.Errorf("from account is not active")
	}

	// Get to account
	var toAccount *repository.AccountRecord
	if req.ToAccountID != uuid.Nil {
		toAccount, err = s.db.GetAccountByID(ctx, req.ToAccountID)
	} else {
		toAccount, err = s.db.GetAccountByNumber(ctx, req.ToAccountNumber)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get to account: %w", err)
	}
	if toAccount == nil {
		return nil, fmt.Errorf("to account not found")
	}
	if toAccount.Status != "ACTIVE" {
		return nil, fmt.Errorf("to account is not active")
	}

	// Calculate fee (internal transfers are free)
	fee := int64(0)
	total := req.Amount + fee

	// Check sufficient balance
	if fromAccount.AvailableBalance < total {
		return nil, fmt.Errorf("insufficient balance")
	}

	// Check daily/monthly limits
	if err := s.checkLimits(ctx, req.FromAccountID, req.Amount); err != nil {
		return nil, err
	}

	// Generate reference
	reference := generateReference("TRF", time.Now())

	// Start database transaction
	dbTx, err := s.db.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer dbTx.Rollback()

	// Create transaction record
	txRecord, err := s.db.CreateTransaction(ctx, dbTx, repository.CreateTransactionInput{
		Reference:      reference,
		IdempotencyKey: req.IdempotencyKey,
		Type:           "TRANSFER",
		Amount:         req.Amount,
		FeeAmount:      fee,
		TotalAmount:    total,
		Currency:       "IDR",
		FromAccountID:  req.FromAccountID,
		ToAccountID:    toAccount.ID,
		Description:    req.Description,
		Metadata: map[string]interface{}{
			"transfer_type": "internal",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Update balances
	if err := s.db.UpdateBalance(ctx, dbTx, req.FromAccountID, -total); err != nil {
		return nil, fmt.Errorf("failed to debit from account: %w", err)
	}
	if err := s.db.UpdateBalance(ctx, dbTx, toAccount.ID, req.Amount); err != nil {
		return nil, fmt.Errorf("failed to credit to account: %w", err)
	}

	// Commit transaction
	if err := dbTx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Update transaction status to completed
	if err := s.db.UpdateTransactionStatus(ctx, dbTx, txRecord.ID, "COMPLETED"); err != nil {
		// Log error but don't fail - status update is not critical
		fmt.Printf("Failed to update transaction status: %v\n", err)
	}

	// Publish events
	s.publishTransferEvents(ctx, req.TraceID, txRecord, fromAccount, toAccount)

	// Cache idempotency key
	if req.IdempotencyKey != "" {
		s.cacheIdempotencyKey(ctx, req.IdempotencyKey, &TransferResponse{
			TransactionID: txRecord.ID,
			Reference:     txRecord.Reference,
			Status:        "COMPLETED",
			Amount:        txRecord.Amount,
			Fee:           txRecord.FeeAmount,
			Total:         txRecord.TotalAmount,
		})
	}

	return &TransferResponse{
		TransactionID: txRecord.ID,
		Reference:     txRecord.Reference,
		Status:        "COMPLETED",
		Amount:        txRecord.Amount,
		Fee:           txRecord.FeeAmount,
		Total:         txRecord.TotalAmount,
	}, nil
}

// ExternalTransfer processes an external transfer (other banks)
func (s *PaymentService) ExternalTransfer(ctx context.Context, req TransferRequest) (*TransferResponse, error) {
	// Check idempotency
	if req.IdempotencyKey != "" {
		existingTx, err := s.checkIdempotency(ctx, req.IdempotencyKey)
		if err != nil {
			return nil, fmt.Errorf("failed to check idempotency: %w", err)
		}
		if existingTx != nil {
			return existingTx, nil
		}
	}

	// Validate external transfer fields
	if req.ToBankCode == "" {
		return nil, fmt.Errorf("bank code is required for external transfers")
	}
	if req.ToAccountNumber == "" {
		return nil, fmt.Errorf("account number is required for external transfers")
	}
	if req.ToAccountName == "" {
		return nil, fmt.Errorf("account name is required for external transfers")
	}

	// Get from account
	fromAccount, err := s.db.GetAccountByID(ctx, req.FromAccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get from account: %w", err)
	}
	if fromAccount == nil {
		return nil, fmt.Errorf("from account not found")
	}
	if fromAccount.Status != "ACTIVE" {
		return nil, fmt.Errorf("from account is not active")
	}

	// Calculate fee for external transfer (e.g., 6500 IDR)
	fee := int64(6500)
	total := req.Amount + fee

	// Check sufficient balance
	if fromAccount.AvailableBalance < total {
		return nil, fmt.Errorf("insufficient balance")
	}

	// Check daily/monthly limits
	if err := s.checkLimits(ctx, req.FromAccountID, req.Amount); err != nil {
		return nil, err
	}

	// Generate reference
	reference := generateReference("EXT", time.Now())

	// Start database transaction
	dbTx, err := s.db.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer dbTx.Rollback()

	// Create transaction record
	txRecord, err := s.db.CreateTransaction(ctx, dbTx, repository.CreateTransactionInput{
		Reference:       reference,
		IdempotencyKey:  req.IdempotencyKey,
		Type:            "TRANSFER",
		Amount:          req.Amount,
		FeeAmount:       fee,
		TotalAmount:     total,
		Currency:        "IDR",
		FromAccountID:   req.FromAccountID,
		ToAccountNumber: req.ToAccountNumber,
		ToBankCode:      req.ToBankCode,
		ToAccountName:   req.ToAccountName,
		Description:     req.Description,
		Metadata: map[string]interface{}{
			"transfer_type": "external",
			"bank_code":     req.ToBankCode,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Debit from account
	if err := s.db.UpdateBalance(ctx, dbTx, req.FromAccountID, -total); err != nil {
		return nil, fmt.Errorf("failed to debit from account: %w", err)
	}

	// Commit transaction
	if err := dbTx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Update transaction status to processing (external transfers are async)
	if err := s.db.UpdateTransactionStatus(ctx, dbTx, txRecord.ID, "PROCESSING"); err != nil {
		fmt.Printf("Failed to update transaction status: %v\n", err)
	}

	// Publish events
	s.publishExternalTransferEvents(ctx, req.TraceID, txRecord, fromAccount)

	// Cache idempotency key
	if req.IdempotencyKey != "" {
		s.cacheIdempotencyKey(ctx, req.IdempotencyKey, &TransferResponse{
			TransactionID: txRecord.ID,
			Reference:     txRecord.Reference,
			Status:        "PROCESSING",
			Amount:        txRecord.Amount,
			Fee:           txRecord.FeeAmount,
			Total:         txRecord.TotalAmount,
		})
	}

	return &TransferResponse{
		TransactionID: txRecord.ID,
		Reference:     txRecord.Reference,
		Status:        "PROCESSING",
		Amount:        txRecord.Amount,
		Fee:           txRecord.FeeAmount,
		Total:         txRecord.TotalAmount,
	}, nil
}

// GetTransactionByReference retrieves a transaction by reference
func (s *PaymentService) GetTransactionByReference(ctx context.Context, reference string) (*repository.TransactionRecord, error) {
	return s.db.GetTransactionByReference(ctx, reference)
}

// GetTransactionsByAccount retrieves transactions for an account
func (s *PaymentService) GetTransactionsByAccount(ctx context.Context, accountID uuid.UUID, limit, offset int) ([]repository.TransactionRecord, error) {
	return s.db.GetTransactionsByAccount(ctx, accountID, limit, offset)
}

// Helper functions

func (s *PaymentService) checkIdempotency(ctx context.Context, key string) (*TransferResponse, error) {
	cacheKey := fmt.Sprintf("idempotency:%s", key)
	data, err := s.rdb.Get(ctx, cacheKey).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var response TransferResponse
	// In production, use proper JSON unmarshaling
	_ = data
	return &response, nil
}

func (s *PaymentService) cacheIdempotencyKey(ctx context.Context, key string, response *TransferResponse) {
	cacheKey := fmt.Sprintf("idempotency:%s", key)
	// Cache for 24 hours
	s.rdb.Set(ctx, cacheKey, response, 24*time.Hour)
}

func (s *PaymentService) checkLimits(ctx context.Context, accountID uuid.UUID, amount int64) error {
	// Check single transaction limit
	if amount > s.cfg.Limits.SingleTransactionLimit {
		return fmt.Errorf("amount exceeds single transaction limit of %d", s.cfg.Limits.SingleTransactionLimit)
	}

	// Check daily limit from cache
	cacheKey := fmt.Sprintf("limits:daily:%s", accountID.String())
	dailyUsed, _ := s.rdb.Get(ctx, cacheKey).Int64()
	if dailyUsed+amount > s.cfg.Limits.DailyTransferLimit {
		return fmt.Errorf("amount exceeds daily transfer limit")
	}

	return nil
}

func (s *PaymentService) publishTransferEvents(ctx context.Context, traceID string, tx *repository.TransactionRecord, fromAccount, toAccount *repository.AccountRecord) {
	metadata := kafka.NewEventMetadata(traceID, "payment-service")

	// Publish TransactionCreated
	s.producer.PublishTransactionCreated(ctx, kafka.TransactionCreatedEvent{
		Metadata:        metadata,
		TransactionID:   tx.ID.String(),
		Reference:       tx.Reference,
		TransactionType: tx.Type,
		Amount:          kafka.Money{Amount: fmt.Sprintf("%d", tx.Amount), Currency: tx.Currency},
		Fee:             kafka.Money{Amount: fmt.Sprintf("%d", tx.FeeAmount), Currency: tx.Currency},
		Total:           kafka.Money{Amount: fmt.Sprintf("%d", tx.TotalAmount), Currency: tx.Currency},
		FromAccount:     kafka.AccountRef{AccountID: fromAccount.ID.String(), AccountNo: fromAccount.AccountNo},
		ToAccount:       kafka.AccountRef{AccountID: toAccount.ID.String(), AccountNo: toAccount.AccountNo},
		Description:     tx.Description,
		Status:          tx.Status,
		CreatedAt:       tx.CreatedAt,
	})

	// Publish AccountDebited
	s.producer.PublishAccountDebited(ctx, kafka.AccountDebitedEvent{
		Metadata:      metadata,
		AccountID:     fromAccount.ID.String(),
		AccountNo:     fromAccount.AccountNo,
		TransactionID: tx.ID.String(),
		Reference:     tx.Reference,
		Amount:        kafka.Money{Amount: fmt.Sprintf("%d", tx.TotalAmount), Currency: tx.Currency},
		BalanceBefore: kafka.Money{Amount: fmt.Sprintf("%d", fromAccount.Balance+tx.TotalAmount), Currency: tx.Currency},
		BalanceAfter:  kafka.Money{Amount: fmt.Sprintf("%d", fromAccount.Balance), Currency: tx.Currency},
		DebitedAt:     time.Now().UTC(),
	})

	// Publish AccountCredited
	s.producer.PublishAccountCredited(ctx, kafka.AccountCreditedEvent{
		Metadata:      metadata,
		AccountID:     toAccount.ID.String(),
		AccountNo:     toAccount.AccountNo,
		TransactionID: tx.ID.String(),
		Reference:     tx.Reference,
		Amount:        kafka.Money{Amount: fmt.Sprintf("%d", tx.Amount), Currency: tx.Currency},
		BalanceBefore: kafka.Money{Amount: fmt.Sprintf("%d", toAccount.Balance-tx.Amount), Currency: tx.Currency},
		BalanceAfter:  kafka.Money{Amount: fmt.Sprintf("%d", toAccount.Balance), Currency: tx.Currency},
		CreditedAt:    time.Now().UTC(),
	})

	// Publish notification request
	s.producer.PublishNotificationRequested(ctx, kafka.NotificationRequestedEvent{
		Metadata:         metadata,
		NotificationID:   uuid.New().String(),
		UserID:           fromAccount.UserID.String(),
		NotificationType: "IN_APP",
		EventCategory:    "TRANSACTION",
		Subject:          "Transfer Successful",
		Body:             fmt.Sprintf("You have successfully transferred %s to %s", formatMoney(tx.Amount, tx.Currency), toAccount.AccountNo),
		RequestedAt:      time.Now().UTC(),
	})
}

func (s *PaymentService) publishExternalTransferEvents(ctx context.Context, traceID string, tx *repository.TransactionRecord, fromAccount *repository.AccountRecord) {
	metadata := kafka.NewEventMetadata(traceID, "payment-service")

	s.producer.PublishTransactionCreated(ctx, kafka.TransactionCreatedEvent{
		Metadata:        metadata,
		TransactionID:   tx.ID.String(),
		Reference:       tx.Reference,
		TransactionType: tx.Type,
		Amount:          kafka.Money{Amount: fmt.Sprintf("%d", tx.Amount), Currency: tx.Currency},
		Fee:             kafka.Money{Amount: fmt.Sprintf("%d", tx.FeeAmount), Currency: tx.Currency},
		Total:           kafka.Money{Amount: fmt.Sprintf("%d", tx.TotalAmount), Currency: tx.Currency},
		FromAccount:     kafka.AccountRef{AccountID: fromAccount.ID.String(), AccountNo: fromAccount.AccountNo},
		Description:     tx.Description,
		Status:          tx.Status,
		CreatedAt:       tx.CreatedAt,
	})
}

func generateReference(prefix string, t time.Time) string {
	return fmt.Sprintf("%s%s%s", prefix, t.Format("20060102150405"), uuid.New().String()[:8])
}

func formatMoney(amount int64, currency string) string {
	return fmt.Sprintf("%s %d", currency, amount)
}
