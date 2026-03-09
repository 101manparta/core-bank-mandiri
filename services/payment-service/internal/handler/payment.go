package handler

import (
	"net/http"
	"strconv"

	"github.com/core-bank-mandiri/payment-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// PaymentHandler handles HTTP requests for payment operations
type PaymentHandler struct {
	paymentService  *service.PaymentService
	transferService *service.TransferService
}

// NewPaymentHandler creates a new payment handler
func NewPaymentHandler(paymentService *service.PaymentService, transferService *service.TransferService) *PaymentHandler {
	return &PaymentHandler{
		paymentService:  paymentService,
		transferService: transferService,
	}
}

// InternalTransfer handles internal transfer requests
// @Summary Process internal transfer
// @Description Transfer funds between accounts within the same bank
// @Tags payments
// @Accept json
// @Produce json
// @Param request body InternalTransferRequest true "Transfer request"
// @Success 200 {object} TransferResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/payments/transfer [post]
func (h *PaymentHandler) InternalTransfer(c *gin.Context) {
	var req InternalTransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "INVALID_REQUEST",
			Message: err.Error(),
		})
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "VALIDATION_ERROR",
			Message: err.Error(),
		})
		return
	}

	// Get user ID from context (set by auth middleware)
	userIDStr := c.GetString("userId")
	if userIDStr == "" {
		userIDStr = c.GetHeader("X-User-Id")
	}

	// Parse from account ID
	fromAccountID, err := uuid.Parse(req.FromAccountID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "INVALID_ACCOUNT_ID",
			Message: "Invalid from account ID",
		})
		return
	}

	// Parse to account ID if provided
	var toAccountID uuid.UUID
	if req.ToAccountID != "" {
		toAccountID, err = uuid.Parse(req.ToAccountID)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Code:    "INVALID_ACCOUNT_ID",
				Message: "Invalid to account ID",
			})
			return
		}
	}

	// Process transfer
	response, err := h.paymentService.InternalTransfer(c.Request.Context(), service.TransferRequest{
		FromAccountID:   fromAccountID,
		ToAccountID:     toAccountID,
		ToAccountNumber: req.ToAccountNumber,
		Amount:          req.Amount,
		Description:     req.Description,
		IdempotencyKey:  c.GetHeader("X-Idempotency-Key"),
		TraceID:         c.GetHeader("X-Request-Id"),
	})

	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "TRANSFER_FAILED",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, TransferResponse{
		Success: true,
		Data: TransferData{
			TransactionID: response.TransactionID.String(),
			Reference:     response.Reference,
			Status:        response.Status,
			Amount:        response.Amount,
			Fee:           response.Fee,
			Total:         response.Total,
		},
	})
}

// ExternalTransfer handles external transfer requests
// @Summary Process external transfer
// @Description Transfer funds to accounts at other banks
// @Tags payments
// @Accept json
// @Produce json
// @Param request body ExternalTransferRequest true "Transfer request"
// @Success 200 {object} TransferResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/payments/transfer/external [post]
func (h *PaymentHandler) ExternalTransfer(c *gin.Context) {
	var req ExternalTransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "INVALID_REQUEST",
			Message: err.Error(),
		})
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "VALIDATION_ERROR",
			Message: err.Error(),
		})
		return
	}

	// Parse from account ID
	fromAccountID, err := uuid.Parse(req.FromAccountID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "INVALID_ACCOUNT_ID",
			Message: "Invalid from account ID",
		})
		return
	}

	// Process transfer
	response, err := h.paymentService.ExternalTransfer(c.Request.Context(), service.TransferRequest{
		FromAccountID:   fromAccountID,
		ToAccountNumber: req.ToAccountNumber,
		ToBankCode:      req.BankCode,
		ToAccountName:   req.AccountName,
		Amount:          req.Amount,
		Description:     req.Description,
		IdempotencyKey:  c.GetHeader("X-Idempotency-Key"),
		TraceID:         c.GetHeader("X-Request-Id"),
	})

	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "TRANSFER_FAILED",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, TransferResponse{
		Success: true,
		Data: TransferData{
			TransactionID: response.TransactionID.String(),
			Reference:     response.Reference,
			Status:        response.Status,
			Amount:        response.Amount,
			Fee:           response.Fee,
			Total:         response.Total,
		},
	})
}

// GetPaymentStatus retrieves the status of a payment
// @Summary Get payment status
// @Description Get the status of a payment by reference
// @Tags payments
// @Produce json
// @Param reference path string true "Payment reference"
// @Success 200 {object} PaymentStatusResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/payments/{reference} [get]
func (h *PaymentHandler) GetPaymentStatus(c *gin.Context) {
	reference := c.Param("reference")
	if reference == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "MISSING_REFERENCE",
			Message: "Payment reference is required",
		})
		return
	}

	tx, err := h.paymentService.GetTransactionByReference(c.Request.Context(), reference)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    "DATABASE_ERROR",
			Message: err.Error(),
		})
		return
	}

	if tx == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Code:    "NOT_FOUND",
			Message: "Payment not found",
		})
		return
	}

	c.JSON(http.StatusOK, PaymentStatusResponse{
		Success: true,
		Data: PaymentStatusData{
			Reference:    tx.Reference,
			Type:         tx.Type,
			Status:       tx.Status,
			Amount:       tx.Amount,
			Fee:          tx.FeeAmount,
			Total:        tx.TotalAmount,
			Currency:     tx.Currency,
			Description:  tx.Description,
			CreatedAt:    tx.CreatedAt,
			ProcessedAt:  tx.ProcessedAt,
		},
	})
}

// GetPaymentHistory retrieves payment history for an account
// @Summary Get payment history
// @Description Get payment history for an account
// @Tags payments
// @Produce json
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} PaymentHistoryResponse
// @Router /api/v1/payments [get]
func (h *PaymentHandler) GetPaymentHistory(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	// Get account ID from query or header
	accountIDStr := c.Query("account_id")
	if accountIDStr == "" {
		accountIDStr = c.GetHeader("X-Account-Id")
	}

	if accountIDStr == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "MISSING_ACCOUNT_ID",
			Message: "Account ID is required",
		})
		return
	}

	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Code:    "INVALID_ACCOUNT_ID",
			Message: "Invalid account ID",
		})
		return
	}

	transactions, err := h.paymentService.GetTransactionsByAccount(c.Request.Context(), accountID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Code:    "DATABASE_ERROR",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, PaymentHistoryResponse{
		Success: true,
		Data:    transactions,
		Meta: PaginationMeta{
			Limit:  limit,
			Offset: offset,
			Total:  len(transactions),
		},
	})
}

// GetTransactionLimits retrieves transaction limits for an account
// @Summary Get transaction limits
// @Description Get transaction limits for an account
// @Tags payments
// @Produce json
// @Success 200 {object} LimitsResponse
// @Router /api/v1/limits [get]
func (h *PaymentHandler) GetTransactionLimits(c *gin.Context) {
	c.JSON(http.StatusOK, LimitsResponse{
		Success: true,
		Data: LimitsData{
			DailyTransferLimit:     100000000,
			DailyWithdrawalLimit:   50000000,
			SingleTransactionLimit: 25000000,
			MonthlyTransferLimit:   1000000000,
		},
	})
}

// GetFeeSchedule retrieves the fee schedule
// @Summary Get fee schedule
// @Description Get the current fee schedule
// @Tags payments
// @Produce json
// @Success 200 {object} FeeScheduleResponse
// @Router /api/v1/fees [get]
func (h *PaymentHandler) GetFeeSchedule(c *gin.Context) {
	c.JSON(http.StatusOK, FeeScheduleResponse{
		Success: true,
		Data: FeeScheduleData{
			InternalTransfer: 0,
			ExternalTransfer: 6500,
			BIFast:           2500,
			RTGS:             25000,
			SKN:              10000,
		},
	})
}

// Request/Response DTOs

type InternalTransferRequest struct {
	FromAccountID string `json:"from_account_id" binding:"required"`
	ToAccountID   string `json:"to_account_id"`
	ToAccountNumber string `json:"to_account_number"`
	Amount        int64  `json:"amount" binding:"required,gt=0"`
	Description   string `json:"description"`
}

func (r *InternalTransferRequest) Validate() error {
	if r.ToAccountID == "" && r.ToAccountNumber == "" {
		return &ValidationError{"Either to_account_id or to_account_number must be provided"}
	}
	if r.Amount <= 0 {
		return &ValidationError{"Amount must be greater than 0"}
	}
	return nil
}

type ExternalTransferRequest struct {
	FromAccountID string `json:"from_account_id" binding:"required"`
	ToAccountNumber string `json:"to_account_number" binding:"required"`
	BankCode      string `json:"bank_code" binding:"required"`
	AccountName   string `json:"account_name" binding:"required"`
	Amount        int64  `json:"amount" binding:"required,gt=0"`
	Description   string `json:"description"`
}

func (r *ExternalTransferRequest) Validate() error {
	if r.Amount <= 0 {
		return &ValidationError{"Amount must be greater than 0"}
	}
	return nil
}

type TransferResponse struct {
	Success bool       `json:"success"`
	Data    TransferData `json:"data"`
}

type TransferData struct {
	TransactionID string `json:"transaction_id"`
	Reference     string `json:"reference"`
	Status        string `json:"status"`
	Amount        int64  `json:"amount"`
	Fee           int64  `json:"fee"`
	Total         int64  `json:"total"`
}

type PaymentStatusResponse struct {
	Success bool              `json:"success"`
	Data    PaymentStatusData `json:"data"`
}

type PaymentStatusData struct {
	Reference   string `json:"reference"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	Amount      int64  `json:"amount"`
	Fee         int64  `json:"fee"`
	Total       int64  `json:"total"`
	Currency    string `json:"currency"`
	Description string `json:"description"`
	CreatedAt   interface{} `json:"created_at"`
	ProcessedAt interface{} `json:"processed_at"`
}

type PaymentHistoryResponse struct {
	Success bool                            `json:"success"`
	Data    interface{}                     `json:"data"`
	Meta    PaginationMeta                  `json:"metadata"`
}

type PaginationMeta struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
}

type LimitsResponse struct {
	Success bool       `json:"success"`
	Data    LimitsData `json:"data"`
}

type LimitsData struct {
	DailyTransferLimit     int64 `json:"daily_transfer_limit"`
	DailyWithdrawalLimit   int64 `json:"daily_withdrawal_limit"`
	SingleTransactionLimit int64 `json:"single_transaction_limit"`
	MonthlyTransferLimit   int64 `json:"monthly_transfer_limit"`
}

type FeeScheduleResponse struct {
	Success bool           `json:"success"`
	Data    FeeScheduleData `json:"data"`
}

type FeeScheduleData struct {
	InternalTransfer int64 `json:"internal_transfer"`
	ExternalTransfer int64 `json:"external_transfer"`
	BIFast           int64 `json:"bi_fast"`
	RTGS             int64 `json:"rtgs"`
	SKN              int64 `json:"skn"`
}

type ErrorResponse struct {
	Success bool   `json:"success"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}
