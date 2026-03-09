package com.corebank.auth.entity;

/**
 * User status enumeration.
 */
public enum UserStatus {
    PENDING,    // Account created but not verified
    ACTIVE,     // Account is active and can be used
    SUSPENDED,  // Account temporarily suspended (too many failed logins, etc.)
    CLOSED      // Account permanently closed
}
