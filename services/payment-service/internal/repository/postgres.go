package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/core-bank-mandiri/payment-service/internal/config"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// PostgresRepository handles database operations
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(cfg config.DatabaseConfig) (*PostgresRepository, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PostgresRepository{db: db}, nil
}

// Close closes the database connection
func (r *PostgresRepository) Close() error {
	return r.db.Close()
}

// Transaction represents a database transaction
type Transaction struct {
	tx *sql.Tx
}

// BeginTx starts a new database transaction
func (r *PostgresRepository) BeginTx(ctx context.Context) (*Transaction, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return &Transaction{tx: tx}, nil
}

// Commit commits the transaction
func (t *Transaction) Commit() error {
	return t.tx.Commit()
}

// Rollback rolls back the transaction
func (t *Transaction) Rollback() error {
	return t.tx.Rollback()
}

// PaymentRepository interface for payment operations
type PaymentRepository interface {
	CreateTransaction(ctx context.Context, tx *Transaction, input CreateTransactionInput) (*TransactionRecord, error)
	GetTransactionByReference(ctx context.Context, reference string) (*TransactionRecord, error)
	GetTransactionsByAccount(ctx context.Context, accountID uuid.UUID, limit, offset int) ([]TransactionRecord, error)
	UpdateTransactionStatus(ctx context.Context, tx *Transaction, id uuid.UUID, status string) error
}

// CreateTransactionInput contains input for creating a transaction
type CreateTransactionInput struct {
	Reference       string
	IdempotencyKey  string
	Type            string
	Amount          int64
	FeeAmount       int64
	TotalAmount     int64
	Currency        string
	FromAccountID   uuid.UUID
	ToAccountID     uuid.UUID
	ToAccountNumber string
	ToBankCode      string
	ToAccountName   string
	Description     string
	Metadata        map[string]interface{}
}

// TransactionRecord represents a transaction record
type TransactionRecord struct {
	ID              uuid.UUID
	Reference       string
	IdempotencyKey  string
	Type            string
	Status          string
	Amount          int64
	FeeAmount       int64
	TotalAmount     int64
	Currency        string
	FromAccountID   uuid.UUID
	ToAccountID     uuid.UUID
	ToAccountNumber string
	ToBankCode      string
	ToAccountName   string
	Description     string
	Metadata        map[string]interface{}
	FailureReason   string
	ProcessedAt     time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Implementation of PaymentRepository
func (r *PostgresRepository) CreateTransaction(ctx context.Context, tx *Transaction, input CreateTransactionInput) (*TransactionRecord, error) {
	query := `
		INSERT INTO transactions (
			id, reference, idempotency_key, transaction_type, status,
			amount, fee_amount, total_amount, currency,
			from_account_id, to_account_id, to_account_number, to_bank_code, to_account_name,
			description, metadata, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		RETURNING id, created_at
	`

	id := uuid.New()
	now := time.Now()

	var createdAt time.Time
	err := tx.tx.QueryRowContext(ctx, query,
		id, input.Reference, input.IdempotencyKey, input.Type, "PENDING",
		input.Amount, input.FeeAmount, input.TotalAmount, input.Currency,
		input.FromAccountID, input.ToAccountID, input.ToAccountNumber, input.ToBankCode, input.ToAccountName,
		input.Description, input.Metadata, now, now,
	).Scan(&id, &createdAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	return &TransactionRecord{
		ID:              id,
		Reference:       input.Reference,
		IdempotencyKey:  input.IdempotencyKey,
		Type:            input.Type,
		Status:          "PENDING",
		Amount:          input.Amount,
		FeeAmount:       input.FeeAmount,
		TotalAmount:     input.TotalAmount,
		Currency:        input.Currency,
		FromAccountID:   input.FromAccountID,
		ToAccountID:     input.ToAccountID,
		ToAccountNumber: input.ToAccountNumber,
		ToBankCode:      input.ToBankCode,
		ToAccountName:   input.ToAccountName,
		Description:     input.Description,
		Metadata:        input.Metadata,
		CreatedAt:       createdAt,
		UpdatedAt:       now,
	}, nil
}

func (r *PostgresRepository) GetTransactionByReference(ctx context.Context, reference string) (*TransactionRecord, error) {
	query := `
		SELECT id, reference, idempotency_key, transaction_type, status,
			amount, fee_amount, total_amount, currency,
			from_account_id, to_account_id, to_account_number, to_bank_code, to_account_name,
			description, metadata, failure_reason, processed_at, created_at, updated_at
		FROM transactions
		WHERE reference = $1
	`

	var record TransactionRecord
	err := r.db.QueryRowContext(ctx, query, reference).Scan(
		&record.ID, &record.Reference, &record.IdempotencyKey, &record.Type, &record.Status,
		&record.Amount, &record.FeeAmount, &record.TotalAmount, &record.Currency,
		&record.FromAccountID, &record.ToAccountID, &record.ToAccountNumber, &record.ToBankCode, &record.ToAccountName,
		&record.Description, &record.Metadata, &record.FailureReason, &record.ProcessedAt, &record.CreatedAt, &record.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	return &record, nil
}

func (r *PostgresRepository) GetTransactionsByAccount(ctx context.Context, accountID uuid.UUID, limit, offset int) ([]TransactionRecord, error) {
	query := `
		SELECT id, reference, idempotency_key, transaction_type, status,
			amount, fee_amount, total_amount, currency,
			from_account_id, to_account_id, to_account_number, to_bank_code, to_account_name,
			description, metadata, failure_reason, processed_at, created_at, updated_at
		FROM transactions
		WHERE from_account_id = $1 OR to_account_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, accountID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions: %w", err)
	}
	defer rows.Close()

	var records []TransactionRecord
	for rows.Next() {
		var record TransactionRecord
		err := rows.Scan(
			&record.ID, &record.Reference, &record.IdempotencyKey, &record.Type, &record.Status,
			&record.Amount, &record.FeeAmount, &record.TotalAmount, &record.Currency,
			&record.FromAccountID, &record.ToAccountID, &record.ToAccountNumber, &record.ToBankCode, &record.ToAccountName,
			&record.Description, &record.Metadata, &record.FailureReason, &record.ProcessedAt, &record.CreatedAt, &record.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}
		records = append(records, record)
	}

	return records, nil
}

func (r *PostgresRepository) UpdateTransactionStatus(ctx context.Context, tx *Transaction, id uuid.UUID, status string) error {
	query := `
		UPDATE transactions
		SET status = $1, updated_at = $2
		WHERE id = $3
	`

	_, err := tx.tx.ExecContext(ctx, query, status, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update transaction status: %w", err)
	}

	return nil
}

// AccountRepository interface for account operations
type AccountRepository interface {
	GetAccountByID(ctx context.Context, id uuid.UUID) (*AccountRecord, error)
	GetAccountByNumber(ctx context.Context, accountNumber string) (*AccountRecord, error)
	UpdateBalance(ctx context.Context, tx *Transaction, id uuid.UUID, amount int64) error
	CheckAccountLimit(ctx context.Context, accountID uuid.UUID, amount int64) (bool, error)
}

// AccountRecord represents an account record
type AccountRecord struct {
	ID               uuid.UUID
	AccountNo        string
	UserID           uuid.UUID
	Type             string
	Status           string
	Balance          int64
	AvailableBalance int64
	Currency         string
}

func (r *PostgresRepository) GetAccountByID(ctx context.Context, id uuid.UUID) (*AccountRecord, error) {
	query := `
		SELECT id, account_no, user_id, account_type, status, balance, available_balance, currency
		FROM accounts
		WHERE id = $1 AND deleted_at IS NULL
	`

	var record AccountRecord
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&record.ID, &record.AccountNo, &record.UserID, &record.Type, &record.Status,
		&record.Balance, &record.AvailableBalance, &record.Currency,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	return &record, nil
}

func (r *PostgresRepository) GetAccountByNumber(ctx context.Context, accountNumber string) (*AccountRecord, error) {
	query := `
		SELECT id, account_no, user_id, account_type, status, balance, available_balance, currency
		FROM accounts
		WHERE account_no = $1 AND deleted_at IS NULL
	`

	var record AccountRecord
	err := r.db.QueryRowContext(ctx, query, accountNumber).Scan(
		&record.ID, &record.AccountNo, &record.UserID, &record.Type, &record.Status,
		&record.Balance, &record.AvailableBalance, &record.Currency,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	return &record, nil
}

func (r *PostgresRepository) UpdateBalance(ctx context.Context, tx *Transaction, id uuid.UUID, amount int64) error {
	query := `
		UPDATE accounts
		SET balance = balance + $1, available_balance = available_balance + $1, updated_at = $2
		WHERE id = $3
	`

	_, err := tx.tx.ExecContext(ctx, query, amount, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update account balance: %w", err)
	}

	return nil
}

func (r *PostgresRepository) CheckAccountLimit(ctx context.Context, accountID uuid.UUID, amount int64) (bool, error) {
	query := `
		SELECT available_balance >= $1
		FROM accounts
		WHERE id = $1 AND deleted_at IS NULL
	`

	var hasSufficientFunds bool
	err := r.db.QueryRowContext(ctx, query, accountID, amount).Scan(&hasSufficientFunds)

	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check account limit: %w", err)
	}

	return hasSufficientFunds, nil
}
