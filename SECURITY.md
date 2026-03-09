# Core Bank Mandiri - Security Documentation

## Executive Summary

This document outlines the comprehensive security architecture and implementation for the Core Bank Mandiri distributed banking system. The security design follows a defense-in-depth approach with multiple layers of protection.

---

## 1. Security Architecture Overview

### 1.1 Defense in Depth Layers

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         SECURITY LAYERS                                  │
├─────────────────────────────────────────────────────────────────────────┤
│  Layer 7: Application Security                                          │
│  - Input validation, Output encoding, CSRF protection                   │
├─────────────────────────────────────────────────────────────────────────┤
│  Layer 6: API Security                                                  │
│  - JWT authentication, Rate limiting, API gateway                       │
├─────────────────────────────────────────────────────────────────────────┤
│  Layer 5: Data Security                                                 │
│  - Encryption at rest, Encryption in transit, Tokenization              │
├─────────────────────────────────────────────────────────────────────────┤
│  Layer 4: Network Security                                              │
│  - Network policies, Security groups, WAF, DDoS protection              │
├─────────────────────────────────────────────────────────────────────────┤
│  Layer 3: Infrastructure Security                                       │
│  - Container security, Pod security policies, Secrets management        │
├─────────────────────────────────────────────────────────────────────────┤
│  Layer 2: Physical Security                                             │
│  - Data center security, Hardware security modules                      │
├─────────────────────────────────────────────────────────────────────────┤
│  Layer 1: Organizational Security                                       │
│  - Access control, Security policies, Incident response                 │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Authentication & Authorization

### 2.1 Authentication Flow

```
┌──────────┐     ┌─────────────┐     ┌─────────────┐     ┌──────────┐
│  Client  │     │ API Gateway │     │ Auth Service│     │  Redis   │
└────┬─────┘     └──────┬──────┘     └──────┬──────┘     └────┬─────┘
     │                  │                   │                  │
     │  1. POST /login  │                   │                  │
     │─────────────────►│                   │                  │
     │                  │  2. Forward       │                  │
     │                  │──────────────────►│                  │
     │                  │                   │  3. Validate     │
     │                  │                   │  Credentials     │
     │                  │                   │─────────────────►│
     │                  │                   │                  │
     │                  │                   │  4. Store Session│
     │                  │                   │◄─────────────────│
     │                  │                   │                  │
     │                  │  5. Generate JWT  │                  │
     │                  │◄──────────────────│                  │
     │  6. Return JWT   │                   │                  │
     │◄─────────────────│                   │                  │
     │                  │                   │                  │
     │  7. Subsequent requests with JWT in Authorization header
     │─────────────────►│                   │                  │
     │                  │  8. Validate JWT  │                  │
     │                  │──────────────────────────────────────►│
     │                  │                   │                  │
```

### 2.2 JWT Token Structure

```json
{
  "header": {
    "alg": "HS256",
    "typ": "JWT"
  },
  "payload": {
    "sub": "user@example.com",
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "sessionId": "session-uuid",
    "iat": 1709712000,
    "exp": 1709712900,
    "iss": "core-bank-mandiri",
    "aud": "core-bank-app"
  }
}
```

### 2.3 Token Security

| Token Type | Expiry | Storage | Refresh |
|------------|--------|---------|---------|
| Access Token | 15 minutes | Memory (client) | Via refresh token |
| Refresh Token | 7 days | Secure HTTP-only cookie | Via re-authentication |

### 2.4 Multi-Factor Authentication (MFA)

**Supported MFA Methods:**
1. TOTP (Time-based One-Time Password) - RFC 6238
2. SMS OTP
3. Email OTP

**MFA Implementation:**
```java
// TOTP Configuration
- Algorithm: SHA1
- Digits: 6
- Period: 30 seconds
- Window: ±1 period (tolerance for clock skew)
- Secret: 160-bit random key
```

---

## 3. Data Security

### 3.1 Encryption at Rest

**Database Encryption:**
```sql
-- PostgreSQL TDE (Transparent Data Encryption)
-- Using pgcrypto for field-level encryption

-- Encrypt sensitive fields
UPDATE users 
SET ssn_encrypted = pgp_sym_encrypt(ssn, 'encryption_key')
WHERE ssn IS NOT NULL;

-- Decrypt when reading
SELECT pgp_sym_decrypt(ssn_encrypted, 'encryption_key') FROM users;
```

**Encryption Standards:**
| Data Type | Algorithm | Key Size |
|-----------|-----------|----------|
| Database | AES-256-GCM | 256 bits |
| Files | AES-256-CBC | 256 bits |
| Backups | AES-256-GCM | 256 bits |

### 3.2 Encryption in Transit

**TLS Configuration:**
```yaml
# Kong Gateway TLS Settings
tls_versions:
  - TLSv1.2
  - TLSv1.3
cipher_suites:
  - TLS_AES_256_GCM_SHA384
  - TLS_CHACHA20_POLY1305_SHA256
  - TLS_AES_128_GCM_SHA256
```

**mTLS for Service-to-Service:**
```
┌─────────────┐     ┌─────────────┐
│   Service A │     │   Service B │
└──────┬──────┘     └──────┬──────┘
       │                   │
       │  1. Client Cert   │
       │──────────────────►│
       │                   │
       │  2. Verify Cert   │
       │  3. Server Cert   │
       │◄──────────────────│
       │                   │
       │  4. Verify Cert   │
       │  5. Encrypted     │
       │◄─────────────────►│
```

### 3.3 Data Classification

| Classification | Examples | Protection |
|----------------|----------|------------|
| Public | Marketing materials | None |
| Internal | Employee directory | Access control |
| Confidential | Account balances | Encryption + Access control |
| Restricted | Passwords, PINs | Strong encryption + MFA |

---

## 4. API Security

### 4.1 Rate Limiting

**Rate Limit Configuration:**
```yaml
# Kong Rate Limiting Plugin
rate_limiting:
  minute: 100
  hour: 1000
  policy: redis
  fault_tolerant: true
  hide_client_headers: false
```

**Rate Limits by Endpoint:**
| Endpoint | Limit | Window |
|----------|-------|--------|
| /auth/login | 5 | minute |
| /auth/register | 3 | hour |
| /payments/transfer | 60 | hour |
| /accounts | 100 | hour |

### 4.2 Input Validation

**Validation Rules:**
```java
// Account Number Validation
@Pattern(regexp = "^[0-9]{10,20}$", message = "Invalid account number")

// Amount Validation
@DecimalMin(value = "0.01", message = "Amount must be positive")
@DecimalMax(value = "1000000000.00", message = "Amount exceeds maximum")

// Email Validation
@Email(message = "Invalid email format")
```

### 4.3 Security Headers

```http
# Response Security Headers
Strict-Transport-Security: max-age=31536000; includeSubDomains
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Content-Security-Policy: default-src 'self'
Referrer-Policy: strict-origin-when-cross-origin
```

---

## 5. Network Security

### 5.1 Network Segmentation

```
┌─────────────────────────────────────────────────────────────────┐
│                         VPC                                      │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                   Public Subnet                         │    │
│  │  ┌─────────────┐  ┌─────────────┐                       │    │
│  │  │   Kong      │  │   WAF       │                       │    │
│  │  │  Gateway    │  │             │                       │    │
│  │  └─────────────┘  └─────────────┘                       │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              │                                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                  Private Subnet                         │    │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐       │    │
│  │  │  Auth   │ │Payment  │ │Transaction│ │ Fraud  │       │    │
│  │  │ Service │ │ Service │ │  Service  │ │Detection│      │    │
│  │  └─────────┘ └─────────┘ └─────────┘ └─────────┘       │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              │                                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                  Data Subnet                            │    │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐      │    │
│  │  │ PostgreSQL  │  │    Redis    │  │    Kafka    │      │    │
│  │  │  Cluster    │  │   Cluster   │  │   Cluster   │      │    │
│  │  └─────────────┘  └─────────────┘  └─────────────┘      │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

### 5.2 Kubernetes Network Policies

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: deny-all-default
  namespace: core-bank
spec:
  podSelector: {}
  policyTypes:
    - Ingress
    - Egress
```

### 5.3 Security Groups

| Group | Inbound | Outbound |
|-------|---------|----------|
| API Gateway | 443 (Internet) | 8080 (Services) |
| Services | 8080 (Gateway) | 5432, 6379, 9092 |
| Database | 5432 (Services) | None |

---

## 6. Secrets Management

### 6.1 Kubernetes Secrets

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: core-bank-secrets
  namespace: core-bank
type: Opaque
stringData:
  DB_PASSWORD: <encrypted>
  JWT_SECRET: <encrypted>
  ENCRYPTION_KEY: <encrypted>
```

### 6.2 Secret Rotation

| Secret Type | Rotation Period | Method |
|-------------|-----------------|--------|
| Database Password | 90 days | Automated |
| JWT Secret | 30 days | Rolling update |
| API Keys | 180 days | Manual + Automated |
| TLS Certificates | 365 days | cert-manager |

---

## 7. Audit & Compliance

### 7.1 Audit Log Requirements

**Logged Events:**
- Authentication (login, logout, MFA)
- Authorization failures
- Data access (read, write, delete)
- Configuration changes
- System events

**Audit Log Format:**
```json
{
  "timestamp": "2024-03-06T10:00:00Z",
  "event_id": "uuid",
  "user_id": "uuid",
  "action": "TRANSFER",
  "resource_type": "TRANSACTION",
  "resource_id": "uuid",
  "old_values": {},
  "new_values": {},
  "ip_address": "192.168.1.1",
  "user_agent": "Mozilla/5.0...",
  "status": "SUCCESS"
}
```

### 7.2 Compliance Standards

| Standard | Requirement | Implementation |
|----------|-------------|----------------|
| PCI-DSS | Card data protection | Tokenization, Encryption |
| GDPR | Data privacy | Right to erasure, Consent |
| SOX | Financial controls | Audit trails, Access control |
| BI-FAST | Indonesian payment standards | Transaction monitoring |

---

## 8. Incident Response

### 8.1 Security Incident Categories

| Category | Examples | Response Time |
|----------|----------|---------------|
| Critical | Data breach, Ransomware | 15 minutes |
| High | DDoS, Service compromise | 1 hour |
| Medium | Failed login attempts | 4 hours |
| Low | Policy violations | 24 hours |

### 8.2 Incident Response Flow

```
┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
│ Detection│───►│ Analysis │───►│Containment│───►│ Recovery │
└──────────┘    └──────────┘    └──────────┘    └──────────┘
     │               │               │               │
     ▼               ▼               ▼               ▼
  Alerts         Triage          Isolate         Restore
  Monitoring     Classification  Mitigate        Verify
```

---

## 9. Security Testing

### 9.1 Testing Schedule

| Test Type | Frequency | Tools |
|-----------|-----------|-------|
| SAST | Every commit | SonarQube, Checkmarx |
| DAST | Weekly | OWASP ZAP, Burp Suite |
| Penetration Testing | Quarterly | External vendor |
| Vulnerability Scanning | Daily | Trivy, Clair |

### 9.2 Security Checklist

**Pre-deployment:**
- [ ] All dependencies scanned for vulnerabilities
- [ ] SAST scan passed
- [ ] Secrets not hardcoded
- [ ] Security headers configured
- [ ] Rate limiting enabled

**Post-deployment:**
- [ ] DAST scan completed
- [ ] Penetration test scheduled
- [ ] Monitoring alerts configured
- [ ] Backup verified

---

## 10. Security Configuration Reference

### 10.1 Spring Security Configuration

```java
@Configuration
@EnableWebSecurity
public class SecurityConfig {
    
    @Bean
    public SecurityFilterChain filterChain(HttpSecurity http) throws Exception {
        http
            .csrf(csrf -> csrf.disable())
            .cors(cors -> cors.configurationSource(corsConfigurationSource()))
            .sessionManagement(session -> 
                session.sessionCreationPolicy(SessionCreationPolicy.STATELESS))
            .authorizeHttpRequests(auth -> auth
                .requestMatchers("/api/v1/auth/**").permitAll()
                .anyRequest().authenticated())
            .addFilterBefore(jwtFilter, UsernamePasswordAuthenticationFilter.class);
        return http.build();
    }
}
```

### 10.2 Password Policy

```yaml
security:
  password:
    min-length: 8
    max-length: 128
    require-uppercase: true
    require-lowercase: true
    require-digit: true
    require-special: true
    history-count: 12  # Cannot reuse last 12 passwords
    max-age-days: 90   # Password expires after 90 days
```

### 10.3 Session Configuration

```yaml
security:
  session:
    timeout-minutes: 30
    max-concurrent: 5
    fixation-protection: migrate
    cookie:
      secure: true
      http-only: true
      same-site: strict
```

---

## 11. Contact & Escalation

### 11.1 Security Team Contacts

| Role | Contact | Availability |
|------|---------|--------------|
| Security Operations | security-ops@corebank.co.id | 24/7 |
| CISO | ciso@corebank.co.id | Business hours |
| Incident Response | incident@corebank.co.id | 24/7 |

### 11.2 Escalation Matrix

| Level | Contact | Time |
|-------|---------|------|
| L1 | Security Operations | Immediate |
| L2 | Security Manager | 30 minutes |
| L3 | CISO | 1 hour |
| L4 | Executive Team | 2 hours |

---

## Appendix A: Security Headers Reference

```http
# Strict Transport Security
Strict-Transport-Security: max-age=63072000; includeSubDomains; preload

# Content Security Policy
Content-Security-Policy: default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'

# X-Frame-Options
X-Frame-Options: DENY

# X-Content-Type-Options
X-Content-Type-Options: nosniff

# Referrer-Policy
Referrer-Policy: strict-origin-when-cross-origin

# Permissions-Policy
Permissions-Policy: geolocation=(), microphone=(), camera=()
```

## Appendix B: OWASP Top 10 Mitigations

| OWASP Risk | Mitigation |
|------------|------------|
| A01: Broken Access Control | RBAC, JWT validation |
| A02: Cryptographic Failures | TLS 1.3, AES-256 |
| A03: Injection | Prepared statements, Input validation |
| A04: Insecure Design | Security by design, Threat modeling |
| A05: Security Misconfiguration | Hardened images, Automated scanning |
| A06: Vulnerable Components | Dependency scanning, SBOM |
| A07: Authentication Failures | MFA, Rate limiting, Account lockout |
| A08: Software & Data Integrity | Code signing, CI/CD security |
| A09: Security Logging | Centralized logging, SIEM |
| A10: SSRF | Network policies, Egress filtering |
