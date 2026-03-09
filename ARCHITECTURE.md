# Core Bank Mandiri - Distributed Banking System Architecture

## Executive Summary

This document describes the architecture of a production-grade distributed banking system designed to handle millions of transactions per day with high availability, fault tolerance, and strong consistency guarantees.

---

## 1. System Architecture Overview

### 1.1 High-Level Architecture Diagram

```
                                    ┌─────────────────────────────────────────────────────────────┐
                                    │                    Kubernetes Cluster                        │
                                    │                                                               │
┌──────────┐                        │  ┌─────────────────────────────────────────────────────────┐ │
│  Client  │                        │  │                    API Gateway (Kong)                    │ │
│  (Web/   │────HTTPS──────────────►│  │  ┌───────────────────────────────────────────────────┐  │ │
│  Mobile) │                        │  │  │              Load Balancer (Nginx)                 │  │ │
└──────────┘                        │  │  └───────────────────────────────────────────────────┘  │ │
                                    │  └─────────────────────────────────────────────────────────┘ │
                                    │                           │                                   │
                                    │         ┌─────────────────┼─────────────────┐                 │
                                    │         │                 │                 │                 │
                                    │         ▼                 ▼                 ▼                 │
                                    │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐           │
                                    │  │   Auth      │  │   Account   │  │   Payment   │           │
                                    │  │  Service    │  │   Service   │  │   Service   │           │
                                    │  │   (Java)    │  │   (Java)    │  │    (Go)     │           │
                                    │  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘           │
                                    │         │                │                │                   │
                                    │         └────────────────┼────────────────┘                   │
                                    │                          │                                    │
                                    │         ┌─────────────────┼─────────────────┐                 │
                                    │         │                 │                 │                 │
                                    │         ▼                 ▼                 ▼                 │
                                    │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐           │
                                    │  │ Transaction │  │    Fraud    │  │ Notification│           │
                                    │  │   Service   │  │  Detection  │  │   Service   │           │
                                    │  │   (Java)    │  │    (Go)     │  │    (Go)     │           │
                                    │  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘           │
                                    │         │                │                │                   │
                                    │         └────────────────┼────────────────┘                   │
                                    │                          │                                    │
                                    │         ┌─────────────────┼─────────────────┐                 │
                                    │         │                 │                 │                 │
                                    │         ▼                 ▼                 ▼                 │
                                    │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐           │
                                    │  │    Audit    │  │   Kafka     │  │   Redis     │           │
                                    │  │   Service   │  │   Cluster   │  │   Cluster   │           │
                                    │  │   (Java)    │  │             │  │             │           │
                                    │  └─────────────┘  └─────────────┘  └─────────────┘           │
                                    │                                                               │
                                    │  ┌─────────────────────────────────────────────────────────┐ │
                                    │  │              PostgreSQL Cluster (Primary + Replicas)    │ │
                                    │  └─────────────────────────────────────────────────────────┘ │
                                    │                                                               │
                                    └─────────────────────────────────────────────────────────────┘
                                                              │
                                    ┌─────────────────────────┼─────────────────────────┐
                                    │                         │                         │
                                    │                         │                         │
                                    ▼                         ▼                         ▼
                            ┌───────────────┐         ┌───────────────┐         ┌───────────────┐
                            │  Prometheus   │         │     ELK       │         │    Grafana    │
                            │               │         │   Stack       │         │               │
                            └───────────────┘         └───────────────┘         └───────────────┘
```

### 1.2 Architecture Principles

| Principle | Implementation |
|-----------|----------------|
| **High Availability** | Multi-replica deployments, automatic failover, health checks |
| **Fault Tolerance** | Circuit breakers, retries with exponential backoff, dead letter queues |
| **Data Consistency** | ACID transactions, distributed locks, eventual consistency where appropriate |
| **Security** | JWT authentication, mTLS, encryption at rest and in transit |
| **Scalability** | Horizontal scaling, stateless services, sharding strategy |

---

## 2. Microservices Design

### 2.1 Service Technology Stack Rationale

| Service | Language | Rationale |
|---------|----------|-----------|
| Auth Service | Java/Spring Boot | Rich security ecosystem, mature OAuth2/OIDC support |
| Account Service | Java/Spring Boot | Complex business logic, JPA/Hibernate for ORM |
| Payment Service | Go | High concurrency, low latency for payment processing |
| Transaction Service | Java/Spring Boot | ACID transactions, Spring's transaction management |
| Fraud Detection | Go | Real-time stream processing, ML integration |
| Notification Service | Go | High throughput, async event processing |
| Audit Service | Java/Spring Boot | Compliance requirements, structured logging |

### 2.2 Service Responsibilities

#### API Gateway (Kong)
- Request routing and load balancing
- Rate limiting (token bucket algorithm)
- JWT validation and authentication enforcement
- Request/response transformation
- API versioning
- Request logging and metrics collection

#### Auth Service
- User authentication (username/password, biometric)
- JWT token generation and validation
- Multi-factor authentication (TOTP, SMS)
- Session management with Redis
- Password reset flow
- OAuth2 client credentials for service-to-service auth

#### Account Service
- Account lifecycle management (create, update, close)
- Balance inquiries (cached via Redis)
- Account holder management
- Account type management (savings, checking, business)
- Interest calculation

#### Payment Service
- Internal transfers (same bank)
- External transfers (RTGS, SKN, BI-Fast)
- Payment validation and limits checking
- Transaction fee calculation
- Payment scheduling

#### Transaction/Ledger Service
- Double-entry bookkeeping
- Atomic balance updates
- Transaction idempotency
- Ledger immutability (append-only)
- Transaction reconciliation

#### Fraud Detection Service
- Real-time transaction scoring
- Pattern detection (velocity, amount, location)
- Machine learning model integration
- Alert generation
- Transaction blocking

#### Notification Service
- Email notifications
- SMS notifications
- Push notifications
- In-app notifications
- Notification preferences

#### Audit Service
- Immutable audit trail
- Compliance reporting
- User activity tracking
- System event logging
- Data retention policies

---

## 3. Database Design

### 3.1 PostgreSQL Schema

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         DATABASE: core_bank                             │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐              │
│  │    users     │    │   accounts   │    │account_holders│             │
│  ├──────────────┤    ├──────────────┤    ├──────────────┤              │
│  │ id           │◄───│ user_id      │    │ id           │              │
│  │ email        │    │ account_no   │◄───│ account_id   │              │
│  │ password_hash│    │ type         │    │ holder_name  │              │
│  │ mfa_secret   │    │ status       │    │ holder_type  │              │
│  │ status       │    │ balance      │    │ percentage   │              │
│  │ created_at   │    │ currency     │    │ created_at   │              │
│  └──────────────┘    └──────────────┘    └──────────────┘              │
│         │                   │                                          │
│         │                   │                                          │
│         ▼                   ▼                                          │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐              │
│  │    sessions  │    │transactions  │    │ledger_entries│              │
│  ├──────────────┤    ├──────────────┤    ├──────────────┤              │
│  │ id           │    │ id           │    │ id           │              │
│  │ user_id      │    │ reference    │    │ transaction_id│             │
│  │ token_hash   │    │ type         │    │ account_id   │              │
│  │ expires_at   │    │ amount       │    │ amount       │              │
│  │ created_at   │    │ status       │    │ direction    │              │
│  └──────────────┘    │ from_account │    │ balance_after│              │
│                      │ to_account   │    │ created_at   │              │
│                      │ created_at   │    └──────────────┘              │
│                      └──────────────┘                                  │
│                              │                                          │
│                              ▼                                          │
│                      ┌──────────────┐    ┌──────────────┐              │
│                      │  audit_logs  │    │ fraud_alerts │              │
│                      ├──────────────┤    ├──────────────┤              │
│                      │ id           │    │ id           │              │
│                      │ user_id      │    │ transaction_id│             │
│                      │ action       │    │ risk_score   │              │
│                      │ resource     │    │ status       │              │
│                      │ metadata     │    │ created_at   │              │
│                      │ created_at   │    └──────────────┘              │
│                      └──────────────┘                                  │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### 3.2 Key Design Decisions

1. **Ledger Immutability**: `ledger_entries` table is append-only with no UPDATE or DELETE operations
2. **Double-Entry Accounting**: Every transaction creates at least two ledger entries (debit and credit)
3. **Soft Deletes**: All entities use `status` field instead of physical deletion
4. **Audit Trail**: All state changes are recorded in `audit_logs`
5. **Idempotency**: Transactions include `reference` field for deduplication

---

## 4. Event-Driven Architecture

### 4.1 Kafka Topics

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         KAFKA CLUSTER                                    │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────────────────┐    ┌─────────────────────────┐             │
│  │  transaction.created    │    │  transaction.completed  │             │
│  │  Partitions: 6          │    │  Partitions: 6          │             │
│  │  Replication: 3         │    │  Replication: 3         │             │
│  └─────────────────────────┘    └─────────────────────────┘             │
│                                                                         │
│  ┌─────────────────────────┐    ┌─────────────────────────┐             │
│  │  account.debited        │    │  account.credited       │             │
│  │  Partitions: 6          │    │  Partitions: 6          │             │
│  │  Replication: 3         │    │  Replication: 3         │             │
│  └─────────────────────────┘    └─────────────────────────┘             │
│                                                                         │
│  ┌─────────────────────────┐    ┌─────────────────────────┐             │
│  │  fraud.alert            │    │  notification.request   │             │
│  │  Partitions: 3          │    │  Partitions: 3          │             │
│  │  Replication: 3         │    │  Replication: 3         │             │
│  └─────────────────────────┘    └─────────────────────────┘             │
│                                                                         │
│  ┌─────────────────────────┐    ┌─────────────────────────┐             │
│  │  audit.event            │    │  user.activity          │             │
│  │  Partitions: 6          │    │  Partitions: 3          │             │
│  │  Replication: 3         │    │  Replication: 3         │             │
│  └─────────────────────────┘    └─────────────────────────┘             │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### 4.2 Event Flow

```
Payment Service          Kafka              Fraud Detection
      │                    │                      │
      │  TransactionCreated│                      │
      ├───────────────────►│                      │
      │                    │                      │
      │                    ├─────────────────────►│  Analyze
      │                    │                      │
      │                    │    FraudAlert        │
      │                    │◄─────────────────────┤  (if suspicious)
      │                    │                      │
      │                    │                      │
      ▼                    │                      │
Transaction Service        │                      │
      │                    │                      │
      │  AccountDebited    │                      │
      ├───────────────────►│                      │
      │                    │                      │
      │  AccountCredited   │                      │
      ├───────────────────►│                      │
      │                    │                      │
      │                    │                      │
      ▼                    ▼                      ▼
                    Notification           Audit Service
                      Service
```

---

## 5. High Availability Design

### 5.1 PostgreSQL High Availability

```
┌─────────────────────────────────────────────────────────────────┐
│                    PostgreSQL Cluster                            │
│                                                                  │
│  ┌─────────────┐                                                 │
│  │   Primary   │───────────────┐                                │
│  │   (Read/    │               │                                │
│  │    Write)   │               │                                │
│  └─────────────┘               │                                │
│         │                      │                                │
│         │  Streaming           │  Synchronous Replication       │
│         │  Replication         │  (for critical data)           │
│         │                      │                                │
│         ▼                      ▼                                │
│  ┌─────────────┐       ┌─────────────┐                          │
│  │  Replica 1  │       │  Replica 2  │                          │
│  │  (Read)     │       │  (Read)     │                          │
│  └─────────────┘       └─────────────┘                          │
│                                                                  │
│  Failover: Patroni + etcd for automatic promotion               │
└─────────────────────────────────────────────────────────────────┘
```

### 5.2 Redis Cluster

```
┌─────────────────────────────────────────────────────────────────┐
│                      Redis Cluster                               │
│                                                                  │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐          │
│  │  Master 1   │    │  Master 2   │    │  Master 3   │          │
│  │  Slots      │    │  Slots      │    │  Slots      │          │
│  │  0-5460     │    │  5461-10922 │    │  10923-16383│          │
│  └──────┬──────┘    └──────┬──────┘    └──────┬──────┘          │
│         │                  │                  │                  │
│         ▼                  ▼                  ▼                  │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐          │
│  │  Replica 1  │    │  Replica 2  │    │  Replica 3  │          │
│  └─────────────┘    └─────────────┘    └─────────────┘          │
│                                                                  │
│  Automatic failover via Redis Sentinel                          │
└─────────────────────────────────────────────────────────────────┘
```

### 5.3 Kubernetes Deployment Strategy

```yaml
# Key HA configurations:
# - PodDisruptionBudget for minimum available pods
# - PodAntiAffinity for spreading across nodes
# - Readiness/Liveness probes
# - Horizontal Pod Autoscaler
# - Rolling updates with maxSurge/maxUnavailable
```

---

## 6. Security Architecture

### 6.1 Defense in Depth

```
┌─────────────────────────────────────────────────────────────────┐
│  Layer 1: Network Security                                       │
│  - VPC isolation                                                 │
│  - Security groups / Network policies                            │
│  - DDoS protection                                               │
├─────────────────────────────────────────────────────────────────┤
│  Layer 2: API Security                                           │
│  - TLS 1.3 for all communications                                │
│  - mTLS for service-to-service                                   │
│  - Rate limiting at gateway                                      │
├─────────────────────────────────────────────────────────────────┤
│  Layer 3: Authentication & Authorization                         │
│  - JWT with short expiry                                         │
│  - OAuth2 for service auth                                       │
│  - RBAC for internal services                                    │
├─────────────────────────────────────────────────────────────────┤
│  Layer 4: Data Security                                          │
│  - Encryption at rest (AES-256)                                  │
│  - Field-level encryption for sensitive data                     │
│  - Key management (HSM/Vault)                                    │
├─────────────────────────────────────────────────────────────────┤
│  Layer 5: Application Security                                   │
│  - Input validation                                              │
│  - SQL injection prevention                                      │
│  - XSS/CSRF protection                                           │
└─────────────────────────────────────────────────────────────────┘
```

### 6.2 Token Flow

```
┌──────────┐     ┌─────────────┐     ┌─────────────┐     ┌──────────┐
│  Client  │────►│ API Gateway │────►│ Auth Service│────►│  Services│
└──────────┘     └─────────────┘     └─────────────┘     └──────────┘
     │                  │                   │                  │
     │  1. Login        │                   │                  │
     │─────────────────►│                   │                  │
     │                  │  2. Forward       │                  │
     │                  │──────────────────►│                  │
     │                  │                   │  3. Validate     │
     │                  │                   │  & Generate JWT  │
     │                  │  4. Return JWT    │                  │
     │                  │◄──────────────────│                  │
     │  5. Return JWT   │                   │                  │
     │◄─────────────────│                   │                  │
     │                  │                   │                  │
     │  6. Subsequent requests with JWT      │                  │
     │─────────────────►│                   │                  │
     │                  │  7. Validate JWT  │                  │
     │                  │──────────────────────────────────────►│
     │                  │                   │                  │
```

---

## 7. Scalability Strategy

### 7.1 Horizontal Scaling

| Component | Scaling Strategy |
|-----------|------------------|
| API Gateway | Stateless, scale based on CPU/memory |
| Services | Stateless, scale based on request queue |
| PostgreSQL | Read replicas for read-heavy workloads |
| Redis | Cluster mode with sharding |
| Kafka | Add brokers and partitions |

### 7.2 Database Sharding Strategy

For accounts table (when needed):
- **Shard Key**: `account_no` (first 2 digits)
- **Shards**: 100 shards (00-99)
- **Routing**: Application-level routing via shard lookup

### 7.3 Caching Strategy

```
┌─────────────────────────────────────────────────────────────────┐
│                      Cache Hierarchy                             │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  L1: Application Cache (Caffeine)                        │    │
│  │  - TTL: 1 minute                                         │    │
│  │  - Account balance (for repeated reads)                  │    │
│  └─────────────────────────────────────────────────────────┘    │
│                            │                                     │
│                            ▼                                     │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  L2: Redis Cache                                         │    │
│  │  - TTL: 5 minutes                                        │    │
│  │  - Account data, session data, OTP                       │    │
│  └─────────────────────────────────────────────────────────┘    │
│                            │                                     │
│                            ▼                                     │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  L3: PostgreSQL (Source of Truth)                        │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

---

## 8. Monitoring & Observability

### 8.1 Metrics Collection (Prometheus)

```
Service Metrics:
- request_count (counter)
- request_duration_seconds (histogram)
- active_connections (gauge)
- error_count (counter)

Database Metrics:
- connection_pool_active (gauge)
- query_duration_seconds (histogram)
- transaction_count (counter)

Business Metrics:
- transactions_per_second (gauge)
- failed_transactions (counter)
- fraud_alerts (counter)
```

### 8.2 Distributed Tracing

```
Request Flow with Trace IDs:

Client Request (trace_id: abc123)
    │
    ▼
API Gateway (span: gateway)
    │
    ▼
Auth Service (span: auth_validate)
    │
    ▼
Payment Service (span: payment_process)
    │
    ├──► Transaction Service (span: transaction_create)
    │
    └──► Kafka (span: event_publish)
```

### 8.3 Alerting Rules

| Alert | Condition | Severity |
|-------|-----------|----------|
| HighErrorRate | error_rate > 1% for 5m | Critical |
| HighLatency | p99_latency > 2s for 5m | Warning |
| DatabaseConnections | pool_exhausted | Critical |
| KafkaLag | consumer_lag > 10000 | Warning |
| FraudSpike | fraud_alerts > 100/hour | Critical |

---

## 9. Deployment Architecture

### 9.1 Environment Strategy

```
┌─────────────────────────────────────────────────────────────────┐
│                                                                  │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │   DEV       │  │   STAGING   │  │  PRODUCTION │              │
│  │             │  │             │  │             │              │
│  │  - 1 replica│  │  - 2 replicas│  │  - 3+ replicas│           │
│  │  - No HA    │  │  - Basic HA │  │  - Full HA  │              │
│  │  - Single AZ│  │  - Multi AZ │  │  - Multi Region│           │
│  └─────────────┘  └─────────────┘  └─────────────┘              │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 9.2 CI/CD Pipeline

```
┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
│   Code   │────│   Build  │────│   Test   │────│  Deploy  │
│  Commit  │    │          │    │          │    │          │
└──────────┘    └──────────┘    └──────────┘    └──────────┘
     │               │               │               │
     │               │               │               │
     ▼               ▼               ▼               ▼
  Git Push      Docker Image    Unit Tests      Kubernetes
                + JAR           Integration     Deployment
                                Tests           + Helm
```

---

## 10. Disaster Recovery

### 10.1 RTO/RPO Targets

| Component | RTO | RPO |
|-----------|-----|-----|
| Core Banking | < 5 min | < 1 min |
| Payment Processing | < 5 min | 0 (synchronous) |
| Notifications | < 15 min | < 5 min |
| Audit Logs | < 30 min | 0 (synchronous) |

### 10.2 Backup Strategy

- **PostgreSQL**: Continuous WAL archiving + daily full backup
- **Redis**: RDB snapshots every hour + AOF
- **Kafka**: MirrorMaker for cross-region replication
- **Configuration**: GitOps with ArgoCD

---

## 11. API Design

### 11.1 REST API Conventions

```
Base URL: /api/v1

Naming:
- Resources: plural nouns (/accounts, /transactions)
- Actions: POST for create, GET for read, PUT/PATCH for update, DELETE for remove
- Versioning: URL path versioning

Response Format:
{
  "success": true,
  "data": { ... },
  "metadata": {
    "request_id": "uuid",
    "timestamp": "ISO8601"
  }
}

Error Format:
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human readable message",
    "details": { ... }
  }
}
```

### 11.2 Key Endpoints

| Service | Method | Endpoint | Description |
|---------|--------|----------|-------------|
| Auth | POST | /auth/login | User login |
| Auth | POST | /auth/logout | User logout |
| Auth | POST | /auth/refresh | Refresh token |
| Accounts | POST | /accounts | Create account |
| Accounts | GET | /accounts/{id} | Get account details |
| Accounts | GET | /accounts | List accounts |
| Payments | POST | /payments/transfer | Transfer funds |
| Payments | GET | /payments/{id} | Get payment status |
| Transactions | GET | /transactions | List transactions |
| Transactions | GET | /transactions/{id} | Get transaction details |

---

## 12. Technology Versions

| Technology | Version | Rationale |
|------------|---------|-----------|
| Java | 21 (LTS) | Latest LTS with virtual threads |
| Spring Boot | 3.2.x | Latest stable with Java 21 support |
| Go | 1.21+ | Latest stable with generics |
| PostgreSQL | 16 | Latest stable with performance improvements |
| Redis | 7.2 | Latest stable with cluster improvements |
| Kafka | 3.6 | Latest stable with KRaft mode |
| Kubernetes | 1.29+ | Latest stable |
| Kong | 3.4 | Latest stable |

---

## 13. Design Decisions Summary

### Why Microservices?
- Independent scaling of services based on load
- Technology diversity (Java for business logic, Go for high-performance)
- Fault isolation
- Team autonomy

### Why Event-Driven?
- Loose coupling between services
- Async processing for non-critical paths
- Audit trail through events
- Easy to add new consumers

### Why PostgreSQL?
- ACID compliance for financial transactions
- Mature ecosystem
- Strong consistency guarantees
- JSONB for flexible metadata

### Why Redis?
- Sub-millisecond latency for sessions
- Built-in data structures for various use cases
- Cluster mode for horizontal scaling

### Why Kafka?
- High throughput for event streaming
- Durability guarantees
- Replay capability for debugging/recovery
- Ecosystem (Connect, Streams)
