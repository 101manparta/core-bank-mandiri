package service

import (
	"context"
	"github.com/core-bank-mandiri/payment-service/internal/config"
	"github.com/core-bank-mandiri/payment-service/internal/kafka"
	"github.com/core-bank-mandiri/payment-service/internal/repository"
	"github.com/redis/go-redis/v9"
)

// TransferService handles transfer operations
type TransferService struct {
	db       *repository.PostgresRepository
	rdb      *redis.Client
	producer *kafka.Producer
	payment  *PaymentService
	cfg      *config.Config
}

// NewTransferService creates a new transfer service
func NewTransferService(
	db *repository.PostgresRepository,
	rdb *redis.Client,
	producer *kafka.Producer,
	payment *PaymentService,
	cfg *config.Config,
) *TransferService {
	return &TransferService{
		db:       db,
		rdb:      rdb,
		producer: producer,
		payment:  payment,
		cfg:      cfg,
	}
}

// InternalTransfer wraps payment service internal transfer
func (s *TransferService) InternalTransfer(ctx context.Context, req TransferRequest) (*TransferResponse, error) {
	return s.payment.InternalTransfer(ctx, req)
}

// ExternalTransfer wraps payment service external transfer
func (s *TransferService) ExternalTransfer(ctx context.Context, req TransferRequest) (*TransferResponse, error) {
	return s.payment.ExternalTransfer(ctx, req)
}
