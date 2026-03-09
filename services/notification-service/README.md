# Notification Service - Core Bank Mandiri

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-Proprietary-red?style=flat)]()
[![Tests](https://img.shields.io/badge/tests-passing-brightgreen?style=flat)]()

Real-time notification delivery service for Core Bank Mandiri banking system. Handles Email, SMS, Push, and In-App notifications.

## 📋 Overview

The Notification Service is responsible for:
- Consuming notification events from Kafka
- Sending email notifications via SMTP
- Sending SMS notifications via SMS gateway (Twilio/Vonage)
- Sending push notifications via FCM/APNS
- Storing in-app notifications
- Managing user notification preferences

## 🏗️ Architecture

```
┌─────────────────┐
│  Payment/Auth/  │
│  Other Services │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│     Kafka       │
│ notification.   │
│ request         │
└────────┬────────┘
         │
         ▼
┌─────────────────┐      ┌──────────────┐
│  Notification   │─────▶│  PostgreSQL  │
│    Service      │      │  (storage)   │
└────────┬────────┘      └──────────────┘
         │
         ├──────▶ SMTP (Email)
         ├──────▶ SMS Gateway
         ├──────▶ FCM/APNS (Push)
         └──────▶ In-App (DB)
```

## 🚀 Quick Start

### Prerequisites

- Go 1.23+
- PostgreSQL 16+
- Kafka 3.6+
- Docker & Docker Compose (optional)

### Local Development

1. **Clone the repository**
```bash
git clone https://github.com/core-bank-mandiri.git
cd core-bank-mandiri/services/notification-service
```

2. **Copy environment file**
```bash
cp .env.example .env
```

3. **Update `.env` with your configuration**
```bash
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=core_bank
POSTGRES_USER=postgres
POSTGRES_PASSWORD=your_password
KAFKA_BROKERS=localhost:9092
SERVER_PORT=8086
```

4. **Run the service**
```bash
go run ./cmd
```

### Docker Deployment

From the project root:
```bash
docker-compose up -d notification-service
```

## 📡 API Endpoints

### Health Check
```bash
GET http://localhost:8086/health
```

**Response:**
```json
{
  "status": "healthy",
  "service": "notification-service"
}
```

### Readiness Check
```bash
GET http://localhost:8086/ready
```

**Response:**
```json
{
  "status": "ready",
  "service": "notification-service"
}
```

## 📨 Kafka Integration

### Consumed Topics

| Topic | Description |
|-------|-------------|
| `notification.request` | Incoming notification requests from other services |

### Event Schema

**Input Event (`notification.request`):**
```json
{
  "metadata": {
    "event_id": "550e8400-e29b-41d4-a716-446655440000",
    "trace_id": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
    "span_id": "6ba7b8109dad",
    "correlation_id": "6ba7b810-9dad-11d1-80b4",
    "occurred_at": "2024-03-06T10:00:00Z",
    "producer": "payment-service",
    "schema_version": "1.0.0"
  },
  "notification_id": "550e8400-e29b-41d4-a716-446655440001",
  "user_id": "550e8400-e29b-41d4-a716-446655440002",
  "notification_type": "EMAIL",
  "event_category": "TRANSACTION",
  "subject": "Transfer Successful",
  "body": "Your transfer has been processed successfully",
  "template_data": {
    "amount": "1000000",
    "currency": "IDR"
  },
  "requested_at": "2024-03-06T10:00:00Z"
}
```

### Notification Types

| Type | Description | Implementation |
|------|-------------|----------------|
| `EMAIL` | Email notifications | SMTP (MailHog for dev) |
| `SMS` | SMS notifications | Twilio/Vonage (placeholder) |
| `PUSH` | Push notifications | FCM/APNS (placeholder) |
| `IN_APP` | In-app notifications | PostgreSQL storage |

### Event Categories

| Category | Description |
|----------|-------------|
| `TRANSACTION` | Transaction-related notifications |
| `SECURITY` | Security alerts (login, MFA, etc.) |
| `MARKETING` | Promotional messages |
| `SYSTEM` | System announcements |

## 🗄️ Database Schema

### notifications Table

```sql
CREATE TABLE notifications (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    notification_type VARCHAR(20) NOT NULL,
    status VARCHAR(20) DEFAULT 'PENDING',
    subject VARCHAR(255),
    body TEXT NOT NULL,
    template_data JSONB,
    sent_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    failed_reason VARCHAR(255),
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);
```

### notification_preferences Table

```sql
CREATE TABLE notification_preferences (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    notification_type VARCHAR(20) NOT NULL,
    event_category VARCHAR(50) NOT NULL,
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
);
```

## ⚙️ Configuration

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `KAFKA_BROKERS` | Kafka broker addresses | `localhost:9092` | No |
| `KAFKA_CONSUMER_GROUP` | Kafka consumer group ID | `notification-service-group` | No |
| `POSTGRES_HOST` | PostgreSQL host | `localhost` | No |
| `POSTGRES_PORT` | PostgreSQL port | `5432` | No |
| `POSTGRES_DB` | Database name | `core_bank` | No |
| `POSTGRES_USER` | Database user | `postgres` | No |
| `POSTGRES_PASSWORD` | Database password | `postgres` | No |
| `SMTP_HOST` | SMTP server host | `mailhog` | No |
| `SMTP_PORT` | SMTP server port | `1025` | No |
| `SMTP_USER` | SMTP username | - | No |
| `SMTP_PASSWORD` | SMTP password | - | No |
| `SMTP_FROM` | From email address | `noreply@corebank.co.id` | No |
| `SERVER_PORT` | HTTP server port | `8086` | No |

## 🧪 Testing

### Unit Tests
```bash
go test -v ./cmd
```

### Run with Coverage
```bash
go test -cover ./cmd
```

### Manual Testing

1. **Start dependencies:**
```bash
docker-compose up -d kafka postgres mailhog
```

2. **Start the service:**
```bash
go run ./cmd
```

3. **Send a test notification via Kafka:**
```bash
# Using kafka-console-producer
docker exec -it kafka kafka-console-producer.sh \
  --broker-list localhost:9092 \
  --topic notification.request
```

4. **Paste the following JSON:**
```json
{
  "metadata": {
    "event_id": "550e8400-e29b-41d4-a716-446655440000",
    "trace_id": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
    "span_id": "6ba7b8109dad",
    "correlation_id": "6ba7b810-9dad-11d1-80b4",
    "occurred_at": "2024-03-06T10:00:00Z",
    "producer": "test-service",
    "schema_version": "1.0.0"
  },
  "notification_id": "550e8400-e29b-41d4-a716-446655440001",
  "user_id": "550e8400-e29b-41d4-a716-446655440002",
  "notification_type": "IN_APP",
  "event_category": "TRANSACTION",
  "subject": "Test Notification",
  "body": "This is a test notification",
  "requested_at": "2024-03-06T10:00:00Z"
}
```

5. **Check logs for output:**
```
Processing notification: 550e8400-e29b-41d4-a716-446655440001, Type: IN_APP, User: 550e8400-e29b-41d4-a716-446655440002
💬 In-app notification saved for user 550e8400-e29b-41d4-a716-446655440002: Test Notification
Notification sent successfully: ... (Type: IN_APP, User: ...)
```

## 🔧 Production Deployment

### SMTP Integration (SendGrid)

Update `sendEmail` method in `main.go`:
```go
func (ns *NotificationService) sendEmail(ctx context.Context, notification *Notification, event NotificationEvent) error {
    // Implement SendGrid integration
    // Or use any SMTP library
}
```

### SMS Integration (Twilio)

Update `sendSMS` method:
```go
func (ns *NotificationService) sendSMS(ctx context.Context, notification *Notification, event NotificationEvent) error {
    // Implement Twilio integration
    // accountSID, authToken from env
}
```

### Push Notification (FCM)

Update `sendPush` method:
```go
func (ns *NotificationService) sendPush(ctx context.Context, notification *Notification, event NotificationEvent) error {
    // Implement Firebase Cloud Messaging
}
```

## 📊 Monitoring

### Metrics to Add (Future)

```go
// Prometheus metrics
var (
    notificationsSent = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "notification_sent_total",
            Help: "Total number of notifications sent",
        },
        []string{"type", "status"},
    )
    
    notificationLatency = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "notification_latency_seconds",
            Help: "Latency of notification delivery",
        },
        []string{"type"},
    )
)
```

### Health Check in Kubernetes

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8086
  initialDelaySeconds: 10
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /ready
    port: 8086
  initialDelaySeconds: 5
  periodSeconds: 5
```

## 🔒 Security

- All database connections use SSL mode (configurable)
- Kafka authentication via SASL (production)
- Environment variables for sensitive data
- No hardcoded credentials

## 📝 Changelog

### v1.0.0 (2024-03-06)
- Initial release
- Kafka consumer integration
- PostgreSQL storage
- Email/SMS/Push/In-App notification support
- User preference management
- Graceful shutdown
- Input validation
- Unit tests

## 🤝 Contributing

1. Create a feature branch
2. Make your changes
3. Add/update tests
4. Run `go test -v ./...`
5. Submit a pull request

## 📄 License

Proprietary software. All rights reserved.

## 📞 Support

- **Technical Issues**: Create an issue in the repository
- **Security Issues**: security@corebank.co.id

---

**Core Bank Mandiri** - Built with ❤️ for Indonesia's financial future
