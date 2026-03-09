package com.corebank.auth.service;

import io.jsonwebtoken.*;
import io.jsonwebtoken.security.Keys;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.security.core.userdetails.UserDetails;
import org.springframework.stereotype.Service;

import javax.crypto.SecretKey;
import java.nio.charset.StandardCharsets;
import java.time.Instant;
import java.util.Date;
import java.util.HashMap;
import java.util.Map;
import java.util.UUID;
import java.util.function.Function;

/**
 * JWT Service for token generation, validation, and parsing.
 * Uses HS256 algorithm with 256-bit secret key.
 */
@Slf4j
@Service
@RequiredArgsConstructor
public class JwtService {

    @Value("${jwt.secret}")
    private String jwtSecret;

    @Value("${jwt.expiration.access-token:900000}")
    private long accessTokenExpiration;

    @Value("${jwt.expiration.refresh-token:604800000}")
    private long refreshTokenExpiration;

    @Value("${jwt.issuer:core-bank-mandiri}")
    private String issuer;

    @Value("${jwt.audience:core-bank-app}")
    private String audience;

    private SecretKey getSigningKey() {
        byte[] keyBytes = jwtSecret.getBytes(StandardCharsets.UTF_8);
        return Keys.hmacShaKeyFor(keyBytes);
    }

    /**
     * Generate access token for authenticated user.
     */
    public String generateAccessToken(UserDetails userDetails, UUID userId, String sessionId) {
        Map<String, Object> claims = new HashMap<>();
        claims.put("sessionId", sessionId);
        claims.put("user_id", userId.toString());

        return Jwts.builder()
                .claims(claims)
                .subject(userDetails.getUsername())
                .issuer(issuer)
                .audience().add(audience).and()
                .issuedAt(Date.from(Instant.now()))
                .expiration(Date.from(Instant.now().plusMillis(accessTokenExpiration)))
                .signWith(getSigningKey())
                .compact();
    }

    /**
     * Generate refresh token.
     */
    public String generateRefreshToken(UserDetails userDetails, UUID userId) {
        return Jwts.builder()
                .subject(userDetails.getUsername())
                .issuer(issuer)
                .audience().add(audience).and()
                .issuedAt(Date.from(Instant.now()))
                .expiration(Date.from(Instant.now().plusMillis(refreshTokenExpiration)))
                .signWith(getSigningKey())
                .compact();
    }

    /**
     * Extract username from token.
     */
    public String extractUsername(String token) {
        return extractClaim(token, Claims::getSubject);
    }

    /**
     * Extract user ID from token.
     */
    public UUID extractUserId(String token) {
        Claims claims = extractAllClaims(token);
        return UUID.fromString(claims.get("user_id", String.class));
    }

    /**
     * Extract session ID from token.
     */
    public String extractSessionId(String token) {
        Claims claims = extractAllClaims(token);
        return claims.get("sessionId", String.class);
    }

    /**
     * Extract expiration date from token.
     */
    public Instant extractExpiration(String token) {
        return extractClaim(token, Claims::getExpiration).toInstant();
    }

    /**
     * Extract a specific claim from token.
     */
    public <T> T extractClaim(String token, Function<Claims, T> claimsResolver) {
        final Claims claims = extractAllClaims(token);
        return claimsResolver.apply(claims);
    }

    /**
     * Extract all claims from token.
     */
    private Claims extractAllClaims(String token) {
        return Jwts.parser()
                .verifyWith(getSigningKey())
                .build()
                .parseSignedClaims(token)
                .getPayload();
    }

    /**
     * Check if token is expired.
     */
    private boolean isTokenExpired(String token) {
        return extractExpiration(token).isBefore(Instant.now());
    }

    /**
     * Validate access token.
     */
    public boolean validateAccessToken(String token, UserDetails userDetails) {
        try {
            final String username = extractUsername(token);
            return (username.equals(userDetails.getUsername()) && !isTokenExpired(token));
        } catch (JwtException e) {
            log.warn("Invalid access token: {}", e.getMessage());
            return false;
        }
    }

    /**
     * Validate refresh token.
     */
    public boolean validateRefreshToken(String token, UserDetails userDetails) {
        try {
            final String username = extractUsername(token);
            return (username.equals(userDetails.getUsername()) && !isTokenExpired(token));
        } catch (JwtException e) {
            log.warn("Invalid refresh token: {}", e.getMessage());
            return false;
        }
    }

    /**
     * Parse and validate token, returning claims if valid.
     */
    public TokenValidationResult validateAndParse(String token) {
        try {
            Claims claims = extractAllClaims(token);
            return TokenValidationResult.valid(claims);
        } catch (ExpiredJwtException e) {
            log.debug("Token expired: {}", e.getMessage());
            return TokenValidationResult.expired();
        } catch (UnsupportedJwtException e) {
            log.debug("Unsupported token: {}", e.getMessage());
            return TokenValidationResult.unsupported();
        } catch (MalformedJwtException e) {
            log.debug("Malformed token: {}", e.getMessage());
            return TokenValidationResult.malformed();
        } catch (SignatureException e) {
            log.debug("Invalid signature: {}", e.getMessage());
            return TokenValidationResult.invalidSignature();
        } catch (IllegalArgumentException e) {
            log.debug("Invalid token: {}", e.getMessage());
            return TokenValidationResult.invalid();
        }
    }

    /**
     * Token validation result holder.
     */
    public record TokenValidationResult(
            boolean isValid,
            boolean isExpired,
            boolean isMalformed,
            boolean isUnsupported,
            boolean isInvalidSignature,
            Claims claims
    ) {
        public static TokenValidationResult valid(Claims claims) {
            return new TokenValidationResult(true, false, false, false, false, claims);
        }

        public static TokenValidationResult expired() {
            return new TokenValidationResult(false, true, false, false, false, null);
        }

        public static TokenValidationResult malformed() {
            return new TokenValidationResult(false, false, true, false, false, null);
        }

        public static TokenValidationResult unsupported() {
            return new TokenValidationResult(false, false, false, true, false, null);
        }

        public static TokenValidationResult invalidSignature() {
            return new TokenValidationResult(false, false, false, false, true, null);
        }

        public static TokenValidationResult invalid() {
            return new TokenValidationResult(false, false, false, false, false, null);
        }
    }
}
