package com.corebank.auth.service;

import com.corebank.auth.entity.Session;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.stereotype.Service;

import java.time.Duration;
import java.util.concurrent.TimeUnit;

/**
 * Redis cache service for session management.
 * Provides fast lookup for session validation.
 */
@Slf4j
@Service
@RequiredArgsConstructor
public class SessionCacheService {

    private static final String SESSION_KEY_PREFIX = "session:";
    private static final Duration DEFAULT_TTL = Duration.ofHours(24);

    private final RedisTemplate<String, Session> redisTemplate;

    /**
     * Save session to cache.
     */
    public void save(String tokenHash, Session session) {
        String key = buildKey(tokenHash);
        redisTemplate.opsForValue().set(key, session, DEFAULT_TTL);
        log.debug("Session cached: {}", key);
    }

    /**
     * Get session from cache.
     */
    public Session get(String tokenHash) {
        String key = buildKey(tokenHash);
        Session session = redisTemplate.opsForValue().get(key);
        log.debug("Session retrieved from cache: {}", key);
        return session;
    }

    /**
     * Invalidate session from cache.
     */
    public void invalidate(String tokenHash) {
        String key = buildKey(tokenHash);
        redisTemplate.delete(key);
        log.debug("Session invalidated: {}", key);
    }

    /**
     * Check if session exists in cache.
     */
    public boolean exists(String tokenHash) {
        String key = buildKey(tokenHash);
        return Boolean.TRUE.equals(redisTemplate.hasKey(key));
    }

    /**
     * Extend session TTL.
     */
    public void touch(String tokenHash) {
        String key = buildKey(tokenHash);
        redisTemplate.expire(key, DEFAULT_TTL, TimeUnit.MILLISECONDS);
    }

    private String buildKey(String tokenHash) {
        return SESSION_KEY_PREFIX + tokenHash;
    }
}
