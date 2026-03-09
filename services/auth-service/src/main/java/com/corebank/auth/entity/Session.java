package com.corebank.auth.entity;

import jakarta.persistence.*;
import lombok.*;
import org.hibernate.annotations.CreationTimestamp;
import org.springframework.data.jpa.domain.support.AuditingEntityListener;

import java.time.Instant;
import java.util.UUID;

/**
 * Session entity for managing user sessions.
 * Sessions are stored in both database and Redis for fast lookup.
 */
@Entity
@Table(name = "sessions")
@EntityListeners(AuditingEntityListener.class)
@Getter
@Setter
@NoArgsConstructor
@AllArgsConstructor
@Builder
public class Session {

    @Id
    @Column(columnDefinition = "UUID")
    private UUID id;

    @Column(nullable = false)
    private UUID userId;

    @Column(name = "token_hash", nullable = false, length = 255)
    private String tokenHash;

    @Column(name = "refresh_token_hash", length = 255)
    private String refreshTokenHash;

    @Column(name = "device_id", length = 255)
    private String deviceId;

    @Column(name = "device_info", columnDefinition = "jsonb")
    private String deviceInfo;

    @Column(name = "ip_address")
    private String ipAddress;

    @Column(name = "user_agent", columnDefinition = "TEXT")
    private String userAgent;

    @Column(name = "expires_at", nullable = false)
    private Instant expiresAt;

    @Column(name = "refreshed_at")
    private Instant refreshedAt;

    @Column(name = "revoked_at")
    private Instant revokedAt;

    @CreationTimestamp
    @Column(name = "created_at", updatable = false)
    private Instant createdAt;

    /**
     * Check if session is valid (not expired and not revoked).
     */
    public boolean isValid() {
        return revokedAt == null && expiresAt.isAfter(Instant.now());
    }

    /**
     * Revoke the session (logout).
     */
    public void revoke() {
        this.revokedAt = Instant.now();
    }

    /**
     * Refresh the session with new expiry.
     */
    public void refresh(Instant newExpiresAt) {
        this.expiresAt = newExpiresAt;
        this.refreshedAt = Instant.now();
    }
}
