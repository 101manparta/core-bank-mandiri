# Core Bank Mandiri - Distributed Banking System

A production-grade, microservices-based distributed banking system designed for high availability, fault tolerance, and horizontal scalability.

## 🏗️ Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Core Bank Mandiri                                  │
│                     Distributed Banking Architecture                         │
└─────────────────────────────────────────────────────────────────────────────┘

                                    ┌─────────────────┐
                                    │   API Gateway   │
                                    │      Kong       │
                                    └────────┬────────┘
                                             │
         ┌───────────────────────────────────┼───────────────────────────────────┐
         │                                   │                                   │
         ▼                                   ▼                                   ▼
┌─────────────────┐              ┌─────────────────┐              ┌─────────────────┐
│   Auth Service  │              │  Account Service│              │  Payment Service│
│     (Java)      │              │     (Java)      │              │      (Go)       │
└────────┬────────┘              └────────┬────────┘              └────────┬────────┘
         │                                │                                │
         └────────────────────────────────┼────────────────────────────────┘
                                          │
         ┌────────────────────────────────┼────────────────────────────────┐
         │                                │                                │
         ▼                                ▼                                ▼
┌─────────────────┐              ┌─────────────────┐              ┌─────────────────┐
│Transaction Svc  │              │  Fraud Detection│              │Notification Svc │
│     (Java)      │              │      (Go)       │              │      (Go)       │
└────────┬────────┘              └────────┬────────┘              └────────┬────────┘
         │                                │                                │
         └────────────────────────────────┼────────────────────────────────┘
                                          │
                                          ▼
                                   ┌─────────────┐
                                   │    Kafka    │
                                   │   Cluster   │
                                   └─────────────┘
```

## 📋 System Requirements

| Component | Technology | Version |
|-----------|------------|---------|
| **Runtime** | Java | 21 (LTS) |
| **Runtime** | Go | 1.21+ |
| **Database** | PostgreSQL | 16 |
| **Cache** | Redis | 7.2 |
| **Message Queue** | Kafka | 3.6 |
| **API Gateway** | Kong | 3.4 |
| **Container** | Docker | 24+ |
| **Orchestration** | Kubernetes | 1.29+ |

## 🚀 Quick Start

### Prerequisites

- Docker & Docker Compose
- Java 21+ (for building Java services)
- Go 1.21+ (for building Go services)
- Maven 3.9+
- Kubernetes (optional, for production deployment)

### Development Setup

1. **Clone the repository**
```bash
git clone https://github.com/core-bank-mandiri.git
cd core-bank-mandiri
```

2. **Start all services with Docker Compose**
```bash
docker-compose up -d
```

3. **Verify services are running**
```bash
docker-compose ps
```

4. **Access the services**
| Service | URL | Description |
|---------|-----|-------------|
| API Gateway | http://localhost:8000 | Main entry point |
| Kong Admin | http://localhost:8001 | Kong administration |
| Auth Service | http://localhost:8081 | Authentication |
| Account Service | http://localhost:8082 | Account management |
| Payment Service | http://localhost:8083 | Payment processing |
| Transaction Service | http://localhost:8084 | Transaction ledger |
| Fraud Detection | http://localhost:8085 | Fraud analysis |
| Notification Service | http://localhost:8086 | Notifications |
| Audit Service | http://localhost:8087 | Audit logging |
| Prometheus | http://localhost:9090 | Metrics |
| Grafana | http://localhost:3000 | Dashboards (admin/admin_password) |
| Kibana | http://localhost:5601 | Log visualization |
| Kafka UI | http://localhost:8090 | Kafka management |
| MailHog | http://localhost:8025 | Email testing |

## 📁 Project Structure

```
core-bank-mandiri/
├── services/
│   ├── auth-service/           # Java/Spring Boot - Authentication
│   ├── account-service/        # Java/Spring Boot - Account management
│   ├── payment-service/        # Go - Payment processing
│   ├── transaction-service/    # Java/Spring Boot - Ledger
│   ├── fraud-detection-service/# Go - Fraud analysis
│   ├── notification-service/   # Go - Notifications
│   └── audit-service/          # Java/Spring Boot - Audit logging
├── infrastructure/
│   ├── docker/                 # Docker configurations
│   ├── kubernetes/             # K8s manifests
│   ├── kong/                   # API Gateway config
│   ├── kafka/                  # Kafka topics setup
│   ├── monitoring/             # Prometheus & Grafana
│   └── logging/                # ELK Stack config
├── database/
│   ├── schema.sql              # Database schema
│   └── migrations/             # Flyway migrations
├── shared/
│   ├── proto/                  # Protocol Buffers
│   └── events/                 # Event schemas
├── docker-compose.yml          # Development orchestration
├── ARCHITECTURE.md             # Architecture documentation
└── SECURITY.md                 # Security documentation
```

## 🔧 Microservices

### 1. Auth Service (Java/Spring Boot)
- User authentication (login/logout)
- JWT token generation and validation
- Multi-factor authentication (TOTP)
- Session management with Redis
- Password management

**Endpoints:**
```
POST   /api/v1/auth/login
POST   /api/v1/auth/logout
POST   /api/v1/auth/refresh
POST   /api/v1/auth/register
POST   /api/v1/auth/mfa/setup
POST   /api/v1/auth/mfa/enable
POST   /api/v1/auth/mfa/verify
```

### 2. Account Service (Java/Spring Boot)
- Account creation and management
- Balance inquiries
- Account holder management
- Account type management

**Endpoints:**
```
POST   /api/v1/accounts
GET    /api/v1/accounts/{id}
GET    /api/v1/accounts
PUT    /api/v1/accounts/{id}
DELETE /api/v1/accounts/{id}
```

### 3. Payment Service (Go)
- Internal transfers (same bank)
- External transfers (RTGS, SKN, BI-Fast)
- Payment validation
- Transaction fee calculation
- Payment scheduling

**Endpoints:**
```
POST   /api/v1/payments/transfer
POST   /api/v1/payments/transfer/external
GET    /api/v1/payments/{reference}
GET    /api/v1/payments
POST   /api/v1/payments/schedule
GET    /api/v1/payments/beneficiaries
```

### 4. Transaction Service (Java/Spring Boot)
- Double-entry bookkeeping
- Atomic balance updates
- Transaction idempotency
- Ledger immutability

**Endpoints:**
```
GET    /api/v1/transactions
GET    /api/v1/transactions/{id}
GET    /api/v1/transactions/account/{accountId}
```

### 5. Fraud Detection Service (Go)
- Real-time transaction scoring
- Pattern detection
- Risk assessment
- Alert generation

**Kafka Topics:**
- `transaction.created` (input)
- `fraud.alert` (output)

### 6. Notification Service (Go)
- Email notifications
- SMS notifications
- Push notifications
- In-app notifications

**Kafka Topics:**
- `notification.request` (input)
- `notification.sent` (output)

### 7. Audit Service (Java/Spring Boot)
- Immutable audit trail
- Compliance reporting
- User activity tracking

**Endpoints:**
```
GET    /api/v1/audit/logs
GET    /api/v1/audit/reports
POST   /api/v1/audit/reports
```

## 📊 Database Schema

### Key Tables

| Table | Description |
|-------|-------------|
| `users` | User accounts and credentials |
| `user_profiles` | User personal information |
| `accounts` | Bank accounts |
| `account_holders` | Joint account holders |
| `transactions` | Transaction records |
| `ledger_entries` | Double-entry ledger |
| `sessions` | User sessions |
| `fraud_alerts` | Fraud detection alerts |
| `audit_logs` | Audit trail |
| `notifications` | Notification records |

### ACID Guarantees

- All financial transactions use database transactions with SERIALIZABLE isolation
- Ledger entries are immutable (append-only)
- Foreign key constraints ensure referential integrity
- Check constraints enforce business rules

## 📨 Event-Driven Architecture

### Kafka Topics

| Topic | Producer | Consumers | Partitions |
|-------|----------|-----------|------------|
| `transaction.created` | Payment Service | Fraud, Transaction, Notification | 6 |
| `transaction.completed` | Transaction Service | Notification, Audit | 6 |
| `account.debited` | Transaction Service | Notification, Analytics | 6 |
| `account.credited` | Transaction Service | Notification, Analytics | 6 |
| `fraud.alert` | Fraud Service | Audit, Dashboard | 3 |
| `notification.request` | Any Service | Notification Service | 3 |
| `audit.event` | All Services | Audit Service, SIEM | 6 |

### Event Schema

Events follow the CloudEvents specification with custom extensions:

```json
{
  "metadata": {
    "event_id": "uuid",
    "trace_id": "uuid",
    "occurred_at": "2024-03-06T10:00:00Z",
    "producer": "payment-service",
    "schema_version": "1.0.0"
  },
  "transaction_id": "uuid",
  "reference": "TRF202403061000001",
  "amount": {
    "amount": "1000000",
    "currency": "IDR"
  },
  "from_account": {
    "account_id": "uuid",
    "account_no": "101234567890"
  },
  "to_account": {
    "account_id": "uuid",
    "account_no": "109876543210"
  }
}
```

## 🔒 Security

### Authentication
- JWT-based authentication
- Short-lived access tokens (15 minutes)
- Refresh tokens (7 days)
- Multi-factor authentication (TOTP)

### Authorization
- Role-based access control (RBAC)
- Service-to-service authentication via mTLS
- API key authentication for partners

### Data Protection
- TLS 1.3 for all communications
- AES-256 encryption at rest
- Field-level encryption for sensitive data
- Automatic secret rotation

### Compliance
- PCI-DSS compliant card handling
- GDPR data privacy controls
- SOX financial audit trails
- BI-FAST payment standards

See [SECURITY.md](SECURITY.md) for detailed security documentation.

## 📈 Monitoring & Observability

### Metrics (Prometheus)

| Metric | Description |
|--------|-------------|
| `http_requests_total` | Total HTTP requests |
| `http_request_duration_seconds` | Request latency |
| `payment_transactions_total` | Payment volume |
| `fraud_alerts_total` | Fraud alerts |
| `jvm_memory_used_bytes` | JVM memory (Java services) |
| `go_goroutines` | Goroutines (Go services) |

### Distributed Tracing (Jaeger)

All requests include trace IDs for end-to-end tracing:
```
Client → Kong → Auth → Payment → Transaction → Kafka
  │        │       │        │          │          │
trace_id: abc123... (propagated across all services)
```

### Logging (ELK Stack)

- Centralized logging via Logstash
- Structured JSON logs
- Log retention: 90 days
- Audit logs: 7 years

## 🚢 Deployment

### Kubernetes Deployment

```bash
# Apply namespace and configurations
kubectl apply -f infrastructure/kubernetes/

# Deploy services
kubectl apply -f infrastructure/kubernetes/deployment.yaml

# Check deployment status
kubectl get pods -n core-bank
```

### Scaling

```bash
# Manual scaling
kubectl scale deployment payment-service --replicas=10 -n core-bank

# HPA is configured for auto-scaling
kubectl get hpa -n core-bank
```

### Rolling Updates

```bash
# Update image
kubectl set image deployment/payment-service \
  payment-service=core-bank-mandiri/payment-service:1.0.1 -n core-bank

# Monitor rollout
kubectl rollout status deployment/payment-service -n core-bank
```

## 🧪 Testing

### Unit Tests

```bash
# Java services
cd services/auth-service && mvn test

# Go services
cd services/payment-service && go test ./...
```

### Integration Tests

```bash
# Start test environment
docker-compose -f docker-compose.test.yml up -d

# Run integration tests
./scripts/run-integration-tests.sh
```

### Load Testing

```bash
# Using k6
k6 run scripts/load-test.js
```

## 📚 API Documentation

API documentation is available via OpenAPI/Swagger:

- Auth Service: http://localhost:8081/swagger-ui.html
- Account Service: http://localhost:8082/swagger-ui.html
- Payment Service: http://localhost:8083/swagger/index.html

## 🔧 Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DB_HOST` | PostgreSQL host | localhost |
| `DB_PORT` | PostgreSQL port | 5432 |
| `DB_NAME` | Database name | core_bank |
| `DB_USERNAME` | Database user | postgres |
| `DB_PASSWORD` | Database password | postgres |
| `REDIS_HOST` | Redis host | localhost |
| `REDIS_PORT` | Redis port | 6379 |
| `KAFKA_BOOTSTRAP_SERVERS` | Kafka brokers | localhost:9092 |
| `JWT_SECRET` | JWT signing key | (required) |

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is proprietary software. All rights reserved.

## 📞 Support


---

**Core Bank Mandiri** - Built with ❤️ for Indonesia's financial future
