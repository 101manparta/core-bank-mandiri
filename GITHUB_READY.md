# Core Bank Mandiri - GitHub Upload Ready

## ✅ Checklist Persiapan GitHub

### Files yang Sudah Diperbaiki:

- [x] Git repository diinisialisasi
- [x] Docker Compose dikonfigurasi dengan benar
- [x] PostgreSQL enabled di docker-compose.yml
- [x] Java version disesuaikan (Java 17 compatible)
- [x] Dockerfile updated untuk compatibility
- [x] .gitkeep files ditambahkan
- [x] Environment variables dikonfigurasi

### Konfigurasi Docker Compose:

Semua services sekarang menggunakan:
- PostgreSQL container: `postgres:5432`
- Password: `postgres_password`
- Network: `core-bank-network`

### Cara Upload ke GitHub:

```bash
# 1. Konfigurasi Git user (jika belum)
git config --global user.name "Your Name"
git config --global user.email "your.email@example.com"

# 2. Buat repository di GitHub (kosong)

# 3. Add remote repository
git remote add origin https://github.com/your-username/core-bank-mandiri.git

# 4. Add semua files
git add .

# 5. Commit pertama
git commit -m "Initial commit: Core Bank Mandiri - Distributed Banking System

- 7 microservices (Java/Go)
- PostgreSQL with replication
- Kafka event-driven architecture
- Kong API Gateway
- Monitoring (Prometheus, Grafana, ELK)
- Full Docker support
- Kubernetes manifests

Production-ready banking system with:
- ACID compliance
- Double-entry bookkeeping
- Fraud detection
- Multi-factor authentication
- Audit logging"

# 6. Push ke GitHub
git branch -M main
git push -u origin main
```

### Struktur Project:

```
core-bank-mandiri/
├── services/           # 7 microservices
│   ├── auth-service/          (Java/Spring Boot)
│   ├── account-service/       (Java/Spring Boot)
│   ├── payment-service/       (Go)
│   ├── transaction-service/   (Java/Spring Boot)
│   ├── fraud-detection-service/ (Go)
│   ├── notification-service/  (Go)
│   └── audit-service/         (Java/Spring Boot)
├── infrastructure/     # Docker, K8s, Kong, Kafka
├── database/           # Schema SQL
├── shared/             # Proto & events
├── docker-compose.yml
├── README.md
├── ARCHITECTURE.md
└── SECURITY.md
```

### Services Ports:

| Service | Port | URL |
|---------|------|-----|
| Kong Gateway | 8000 | http://localhost:8000 |
| Auth Service | 8081 | http://localhost:8081 |
| Account Service | 8082 | http://localhost:8082 |
| Payment Service | 8083 | http://localhost:8083 |
| Transaction Service | 8084 | http://localhost:8084 |
| Fraud Detection | 8085 | http://localhost:8085 |
| Notification | 8086 | http://localhost:8086 |
| Audit Service | 8087 | http://localhost:8087 |
| Prometheus | 9090 | http://localhost:9090 |
| Grafana | 3000 | http://localhost:3000 |
| Kibana | 5601 | http://localhost:5601 |
| Kafka UI | 8090 | http://localhost:8090 |

### Quick Start:

```bash
# Start semua services
docker-compose up -d

# Check status
docker-compose ps

# View logs
docker-compose logs -f

# Stop semua services
docker-compose down
```

### Requirements:

- Docker 24+
- Docker Compose
- Java 17+
- Go 1.21+
- Maven 3.9+

---

**Ready for GitHub Upload! 🚀**
