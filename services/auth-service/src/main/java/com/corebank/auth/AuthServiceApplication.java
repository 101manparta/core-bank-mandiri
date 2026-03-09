package com.corebank.auth;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.boot.autoconfigure.domain.EntityScan;
import org.springframework.cache.annotation.EnableCaching;
import org.springframework.data.jpa.repository.config.EnableJpaAuditing;
import org.springframework.data.jpa.repository.config.EnableJpaRepositories;
import org.springframework.kafka.annotation.EnableKafka;
import org.springframework.scheduling.annotation.EnableAsync;
import org.springframework.scheduling.annotation.EnableScheduling;

/**
 * Auth Service - Core Bank Mandiri
 * 
 * Responsible for:
 * - User authentication (login/logout)
 * - JWT token generation and validation
 * - Multi-factor authentication (TOTP)
 * - Session management
 * - Password management
 */
@SpringBootApplication
@EnableCaching
@EnableKafka
@EnableAsync
@EnableScheduling
@EnableJpaAuditing
@EnableJpaRepositories(basePackages = "com.corebank.auth.repository")
@EntityScan(basePackages = "com.corebank.auth.entity")
public class AuthServiceApplication {

    public static void main(String[] args) {
        SpringApplication.run(AuthServiceApplication.class, args);
    }
}
