package com.corebank.auth.repository;

import com.corebank.auth.entity.User;
import com.corebank.auth.entity.UserStatus;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Lock;
import org.springframework.data.jpa.repository.Modifying;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;

import jakarta.persistence.LockModeType;
import java.time.Instant;
import java.util.Optional;
import java.util.UUID;

/**
 * Repository for User entity with pessimistic locking support.
 */
@Repository
public interface UserRepository extends JpaRepository<User, UUID> {

    /**
     * Find user by email with pessimistic write lock.
     * Used for login to prevent race conditions.
     */
    @Lock(LockModeType.PESSIMISTIC_WRITE)
    @Query("SELECT u FROM User u WHERE u.email = :email")
    Optional<User> findByEmailWithLock(@Param("email") String email);

    /**
     * Find user by email without lock (for read operations).
     */
    Optional<User> findByEmail(String email);

    /**
     * Find user by phone number.
     */
    Optional<User> findByPhoneNumber(String phoneNumber);

    /**
     * Check if email exists.
     */
    boolean existsByEmail(String email);

    /**
     * Check if phone number exists.
     */
    boolean existsByPhoneNumber(String phoneNumber);

    /**
     * Count users by status.
     */
    long countByStatus(UserStatus status);

    /**
     * Update failed login attempts and lock status.
     */
    @Modifying
    @Query("UPDATE User u SET u.failedLoginAttempts = :attempts, u.lockedUntil = :lockedUntil, u.status = :status WHERE u.id = :id")
    void updateLoginAttempts(@Param("id") UUID id, 
                             @Param("attempts") int attempts,
                             @Param("lockedUntil") Instant lockedUntil,
                             @Param("status") UserStatus status);

    /**
     * Reset failed login attempts on successful login.
     */
    @Modifying
    @Query("UPDATE User u SET u.failedLoginAttempts = 0, u.lockedUntil = null, u.lastLoginAt = :lastLoginAt WHERE u.id = :id")
    void resetLoginAttempts(@Param("id") UUID id, @Param("lastLoginAt") Instant lastLoginAt);

    /**
     * Find users with expired lock.
     */
    @Query("SELECT u FROM User u WHERE u.status = 'SUSPENDED' AND u.lockedUntil < :now")
    Iterable<User> findUsersWithExpiredLock(@Param("now") Instant now);
}
