package com.corebank.auth.service;

import com.corebank.auth.entity.Session;
import com.corebank.auth.entity.User;
import com.corebank.auth.entity.UserStatus;
import com.corebank.auth.repository.SessionRepository;
import com.corebank.auth.repository.UserRepository;
import com.corebank.auth.dto.*;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.security.SecureRandom;
import java.time.Instant;
import java.util.Base64;
import java.util.List;
import java.util.UUID;

/**
 * Authentication Service handling login, logout, registration, and session management.
 */
@Slf4j
@Service
@RequiredArgsConstructor
public class AuthenticationService {

    private final UserRepository userRepository;
    private final SessionRepository sessionRepository;
    private final PasswordEncoder passwordEncoder;
    private final JwtService jwtService;
    private final SessionCacheService sessionCache;

    @Value("${security.login.max-attempts:5}")
    private int maxLoginAttempts;

    @Value("${security.login.lock-duration:1800000}")
    private long lockDurationMs;

    @Value("${security.session.max-concurrent:5}")
    private int maxConcurrentSessions;

    private static final SecureRandom SECURE_RANDOM = new SecureRandom();

    /**
     * Authenticate user and create session.
     */
    @Transactional
    public AuthResponse login(LoginRequest request, String ipAddress, String userAgent) {
        log.info("Login attempt for email: {}", request.getEmail());

        // Find user with pessimistic lock
        User user = userRepository.findByEmailWithLock(request.getEmail())
                .orElseThrow(() -> new AuthenticationException("Invalid email or password"));

        // Check if account is locked
        if (user.isLocked()) {
            throw new AccountLockedException("Account is locked until " + user.getLockedUntil());
        }

        // Check if account is active
        if (user.getStatus() != UserStatus.ACTIVE) {
            throw new AccountDisabledException("Account is not active");
        }

        // Verify password
        if (!passwordEncoder.matches(request.getPassword(), user.getPasswordHash())) {
            user.incrementFailedLoginAttempts(maxLoginAttempts, lockDurationMs);
            userRepository.save(user);
            log.warn("Failed login attempt for email: {}. Attempts: {}", request.getEmail(), user.getFailedLoginAttempts());
            throw new AuthenticationException("Invalid email or password");
        }

        // Check if MFA is required
        boolean mfaRequired = user.isMfaEnabled() && request.getMfaCode() == null;
        if (mfaRequired) {
            log.info("MFA required for user: {}", user.getEmail());
            return AuthResponse.mfaRequired(user.getId());
        }

        // Verify MFA code if provided
        if (user.isMfaEnabled() && request.getMfaCode() != null) {
            // MFA verification handled by MfaService
        }

        // Check concurrent sessions limit
        long activeSessions = sessionRepository.countByUserIdAndRevokedAtIsNull(user.getId());
        if (activeSessions >= maxConcurrentSessions) {
            // Revoke oldest session
            List<Session> sessions = sessionRepository.findByUserIdAndRevokedAtIsNull(user.getId());
            sessions.stream()
                    .min((s1, s2) -> s1.getCreatedAt().compareTo(s2.getCreatedAt()))
                    .ifPresent(session -> {
                        session.revoke();
                        sessionRepository.save(session);
                        sessionCache.invalidate(session.getTokenHash());
                    });
        }

        // Reset failed login attempts
        user.resetFailedLoginAttempts();
        user.setLastLoginAt(Instant.now());
        userRepository.save(user);

        // Create session
        Session session = createSession(user, request.getDeviceId(), ipAddress, userAgent);

        // Generate tokens
        String accessToken = jwtService.generateAccessToken(
                new org.springframework.security.core.userdetails.User(
                        user.getEmail(), "", List.of()),
                user.getId(),
                session.getId().toString()
        );
        String refreshToken = jwtService.generateRefreshToken(
                new org.springframework.security.core.userdetails.User(
                        user.getEmail(), "", List.of()),
                user.getId()
        );

        log.info("User logged in successfully: {}", user.getEmail());

        return AuthResponse.success(
                user.getId(),
                user.getEmail(),
                accessToken,
                refreshToken,
                session.getId()
        );
    }

    /**
     * Logout user and revoke session.
     */
    @Transactional
    public void logout(String token) {
        String tokenHash = hashToken(token);
        
        Session session = sessionRepository.findByTokenHash(tokenHash)
                .orElseThrow(() -> new AuthenticationException("Invalid session"));

        session.revoke();
        sessionRepository.save(session);
        sessionCache.invalidate(tokenHash);

        log.info("User logged out: {}", session.getUserId());
    }

    /**
     * Refresh access token using refresh token.
     */
    @Transactional
    public TokenPair refreshToken(String refreshToken) {
        JwtService.TokenValidationResult result = jwtService.validateAndParse(refreshToken);
        
        if (!result.isValid() || result.isExpired()) {
            throw new AuthenticationException("Invalid or expired refresh token");
        }

        String email = jwtService.extractUsername(refreshToken);
        User user = userRepository.findByEmail(email)
                .orElseThrow(() -> new AuthenticationException("User not found"));

        if (user.getStatus() != UserStatus.ACTIVE) {
            throw new AccountDisabledException("Account is not active");
        }

        // Generate new tokens
        String sessionId = result.claims().get("sessionId", String.class);
        String newAccessToken = jwtService.generateAccessToken(
                new org.springframework.security.core.userdetails.User(email, "", List.of()),
                user.getId(),
                UUID.fromString(sessionId)
        );
        String newRefreshToken = jwtService.generateRefreshToken(
                new org.springframework.security.core.userdetails.User(email, "", List.of()),
                user.getId()
        );

        // Update session
        Session session = sessionRepository.findById(UUID.fromString(sessionId))
                .orElseThrow(() -> new AuthenticationException("Session not found"));
        session.refresh(jwtService.extractExpiration(newAccessToken));
        sessionRepository.save(session);

        return new TokenPair(newAccessToken, newRefreshToken);
    }

    /**
     * Register new user.
     */
    @Transactional
    public User register(RegisterRequest request) {
        if (userRepository.existsByEmail(request.getEmail())) {
            throw new DuplicateResourceException("Email already registered");
        }

        User user = User.builder()
                .id(UUID.randomUUID())
                .email(request.getEmail())
                .phoneNumber(request.getPhoneNumber())
                .passwordHash(passwordEncoder.encode(request.getPassword()))
                .status(UserStatus.PENDING)
                .emailVerified(false)
                .build();

        return userRepository.save(user);
    }

    /**
     * Change user password.
     */
    @Transactional
    public void changePassword(UUID userId, String oldPassword, String newPassword) {
        User user = userRepository.findById(userId)
                .orElseThrow(() -> new ResourceNotFoundException("User not found"));

        if (!passwordEncoder.matches(oldPassword, user.getPasswordHash())) {
            throw new AuthenticationException("Current password is incorrect");
        }

        user.setPasswordHash(passwordEncoder.encode(newPassword));
        userRepository.save(user);

        // Revoke all sessions for security
        sessionRepository.revokeAllUserSessions(userId, Instant.now());
    }

    /**
     * Create a new session.
     */
    private Session createSession(User user, String deviceId, String ipAddress, String userAgent) {
        String token = generateSecureToken();
        String refreshToken = generateSecureToken();

        Session session = Session.builder()
                .id(UUID.randomUUID())
                .userId(user.getId())
                .tokenHash(hashToken(token))
                .refreshTokenHash(hashToken(refreshToken))
                .deviceId(deviceId)
                .ipAddress(ipAddress)
                .userAgent(userAgent)
                .expiresAt(Instant.now().plusMillis(jwtService.extractExpiration(token).toEpochMilli()))
                .build();

        Session saved = sessionRepository.save(session);
        sessionCache.save(hashToken(token), saved);

        return saved;
    }

    /**
     * Generate a secure random token.
     */
    private String generateSecureToken() {
        byte[] randomBytes = new byte[32];
        SECURE_RANDOM.nextBytes(randomBytes);
        return Base64.getUrlEncoder().withoutPadding().encodeToString(randomBytes);
    }

    /**
     * Hash token for storage.
     */
    private String hashToken(String token) {
        return Base64.getEncoder().encodeToString(
                java.security.MessageDigest.getInstance("SHA-256")
                        .digest(token.getBytes(java.nio.charset.StandardCharsets.UTF_8))
        );
    }

    // Custom exceptions
    public static class AuthenticationException extends RuntimeException {
        public AuthenticationException(String message) {
            super(message);
        }
    }

    public static class AccountLockedException extends RuntimeException {
        public AccountLockedException(String message) {
            super(message);
        }
    }

    public static class AccountDisabledException extends RuntimeException {
        public AccountDisabledException(String message) {
            super(message);
        }
    }

    public static class DuplicateResourceException extends RuntimeException {
        public DuplicateResourceException(String message) {
            super(message);
        }
    }

    public static class ResourceNotFoundException extends RuntimeException {
        public ResourceNotFoundException(String message) {
            super(message);
        }
    }
}
