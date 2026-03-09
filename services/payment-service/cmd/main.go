package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/core-bank-mandiri/payment-service/internal/config"
	"github.com/core-bank-mandiri/payment-service/internal/handler"
	"github.com/core-bank-mandiri/payment-service/internal/kafka"
	"github.com/core-bank-mandiri/payment-service/internal/repository"
	"github.com/core-bank-mandiri/payment-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// Payment Service - Core Bank Mandiri
// High-performance payment processing service built with Go

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database connection
	db, err := repository.NewPostgresRepository(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize Redis connection
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       0,
	})

	// Initialize Kafka producer
	kafkaProducer, err := kafka.NewProducer(cfg.Kafka)
	if err != nil {
		log.Fatalf("Failed to create Kafka producer: %v", err)
	}
	defer kafkaProducer.Close()

	// Initialize services
	paymentService := service.NewPaymentService(db, rdb, kafkaProducer, cfg)
	transferService := service.NewTransferService(db, rdb, kafkaProducer, paymentService, cfg)

	// Initialize handlers
	paymentHandler := handler.NewPaymentHandler(paymentService, transferService)

	// Setup Gin router
	router := setupRouter(paymentHandler, cfg)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Starting Payment Service on port %s", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited gracefully")
}

func setupRouter(paymentHandler *handler.PaymentHandler, cfg *config.Config) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	// Global middleware
	router.Use(gin.Recovery())
	router.Use(loggingMiddleware())
	router.Use(corsMiddleware(cfg))

	// Health check endpoints
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "payment-service",
			"timestamp": time.Now().UTC(),
		})
	})

	router.GET("/ready", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ready",
			"service": "payment-service",
		})
	})

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Payment routes
		payments := v1.Group("/payments")
		{
			// Internal transfer (same bank)
			payments.POST("/transfer", paymentHandler.InternalTransfer)

			// External transfer (other banks)
			payments.POST("/transfer/external", paymentHandler.ExternalTransfer)

			// Payment status
			payments.GET("/:reference", paymentHandler.GetPaymentStatus)

			// Payment history
			payments.GET("/", paymentHandler.GetPaymentHistory)
		}

		// Limits and fees
		v1.GET("/limits", paymentHandler.GetTransactionLimits)
		v1.GET("/fees", paymentHandler.GetFeeSchedule)
	}

	return router
}

func loggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()

		log.Printf("[%d] %s %s %v", statusCode, c.Request.Method, path, latency)
	}
}

func corsMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", cfg.CORS.AllowOrigins)
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-Id, X-Idempotency-Key")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "3600")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
