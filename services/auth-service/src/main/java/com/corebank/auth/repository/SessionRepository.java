package com.corebank.auth.repository;

import com.corebank.auth.entity.Session;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Modifying;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;

import java.time.Instant;
import java.util.List;
import java.util.Optional;
import java.util.UUID;

/**
 * Repository for Session entity.
 */
@Repository
public interface SessionRepository extends JpaRepository<Session, UUID> {

    /**
     * Find session by token hash.
     */
    Optional<Session> findByTokenHash(String tokenHash);

    /**
     * Find all active sessions for a user.
     */
    List<Session> findByUserIdAndRevokedAtIsNull(UUID userId);

    /**
     * Count active sessions for a user.
     */
    long countByUserIdAndRevokedAtIsNull(UUID userId);

    /**
     * Find sessions expiring soon.
     */
    @Query("SELECT s FROM Session s WHERE s.expiresAt < :threshold AND s.revokedAt IS NULL")
    List<Session> findSessionsExpiringBefore(@Param("threshold") Instant threshold);

    /**
     * Revoke all sessions for a user (force logout everywhere).
     */
    @Modifying
    @Query("UPDATE Session s SET s.revokedAt = :revokedAt WHERE s.userId = :userId AND s.revokedAt IS NULL")
    void revokeAllUserSessions(@Param("userId") UUID userId, @Param("revokedAt") Instant revokedAt);

    /**
     * Delete expired sessions.
     */
    @Modifying
    @Query("DELETE FROM Session s WHERE s.expiresAt < :now")
    void deleteExpiredSessions(@Param("now") Instant now);

    /**
     * Find session by device ID for a user.
     */
    Optional<Session> findByUserIdAndDeviceId(UUID userId, String deviceId);
}
