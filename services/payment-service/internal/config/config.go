package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the payment service
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Kafka    KafkaConfig
	CORS     CORSConfig
	Limits   LimitsConfig
}

type ServerConfig struct {
	Port              string
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
}

type DatabaseConfig struct {
	Host            string
	Port            string
	Name            string
	User            string
	Password        string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type KafkaConfig struct {
	Brokers        []string
	TopicPrefix    string
	Compression    string
	BatchSize      int
	BatchTimeout   time.Duration
}

type CORSConfig struct {
	AllowOrigins string
}

type LimitsConfig struct {
	DailyTransferLimit     int64
	DailyWithdrawalLimit   int64
	SingleTransactionLimit int64
	MonthlyTransferLimit   int64
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port:            getEnv("SERVER_PORT", "8080"),
			ReadTimeout:     getDurationEnv("SERVER_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:    getDurationEnv("SERVER_WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:     getDurationEnv("SERVER_IDLE_TIMEOUT", 60*time.Second),
			ShutdownTimeout: getDurationEnv("SERVER_SHUTDOWN_TIMEOUT", 30*time.Second),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "5432"),
			Name:            getEnv("DB_NAME", "core_bank"),
			User:            getEnv("DB_USERNAME", "postgres"),
			Password:        getEnv("DB_PASSWORD", "postgres"),
			MaxOpenConns:    getIntEnv("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getIntEnv("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getDurationEnv("DB_CONN_MAX_LIFETIME", 30*time.Minute),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getIntEnv("REDIS_DB", 0),
		},
		Kafka: KafkaConfig{
			Brokers:      getEnvSlice("KAFKA_BROKERS", []string{"localhost:9092"}),
			TopicPrefix:  getEnv("KAFKA_TOPIC_PREFIX", "corebank"),
			Compression:  getEnv("KAFKA_COMPRESSION", "lz4"),
			BatchSize:    getIntEnv("KAFKA_BATCH_SIZE", 100),
			BatchTimeout: getDurationEnv("KAFKA_BATCH_TIMEOUT", 100*time.Millisecond),
		},
		CORS: CORSConfig{
			AllowOrigins: getEnv("CORS_ALLOW_ORIGINS", "*"),
		},
		Limits: LimitsConfig{
			DailyTransferLimit:     getInt64Env("LIMIT_DAILY_TRANSFER", 100000000),
			DailyWithdrawalLimit:   getInt64Env("LIMIT_DAILY_WITHDRAWAL", 50000000),
			SingleTransactionLimit: getInt64Env("LIMIT_SINGLE_TRANSACTION", 25000000),
			MonthlyTransferLimit:   getInt64Env("LIMIT_MONTHLY_TRANSFER", 1000000000),
		},
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getInt64Env(key string, defaultValue int64) int64 {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value, exists := os.LookupEnv(key); exists {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getEnvSlice(key string, defaultValue []string) []string {
	if value, exists := os.LookupEnv(key); exists {
		if value != "" {
			return splitString(value, ",")
		}
	}
	return defaultValue
}

func splitString(s, sep string) []string {
	var result []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
			i = start - 1
		}
	}
	return result
}

func (c *Config) DatabaseURL() string {
	return fmt.Sprintf("postgres://%s:%s@[%s]:%s/%s?sslmode=disable",
		c.Database.User,
		c.Database.Password,
		c.Database.Host,
		c.Database.Port,
		c.Database.Name,
	)
}

func (c *Config) RedisAddr() string {
	return fmt.Sprintf("%s:%s", c.Redis.Host, c.Redis.Port)
}
