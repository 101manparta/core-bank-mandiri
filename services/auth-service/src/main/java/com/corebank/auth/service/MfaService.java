package com.corebank.auth.service;

import com.corebank.auth.entity.User;
import com.corebank.auth.repository.UserRepository;
import dev.samstevens.totp.code.*;
import dev.samstevens.totp.exceptions.QrGenerationException;
import dev.samstevens.totp.qr.QrData;
import dev.samstevens.totp.qr.QrGenerator;
import dev.samstevens.totp.qr.ZxingPngQrGenerator;
import dev.samstevens.totp.secret.DefaultSecretGenerator;
import dev.samstevens.totp.secret.SecretGenerator;
import dev.samstevens.totp.time.SystemTimeProvider;
import dev.samstevens.totp.time.TimeProvider;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.util.UUID;

/**
 * MFA Service for TOTP-based multi-factor authentication.
 * Uses RFC 6238 TOTP standard.
 */
@Slf4j
@Service
@RequiredArgsConstructor
public class MfaService {

    private final UserRepository userRepository;

    @Value("${security.mfa.issuer:CoreBankMandiri}")
    private String issuer;

    @Value("${security.mfa.window:1}")
    private int timeWindow;

    private final SecretGenerator secretGenerator = new DefaultSecretGenerator();
    private final TimeProvider timeProvider = new SystemTimeProvider();
    private final QrGenerator qrGenerator = new ZxingPngQrGenerator();

    /**
     * Generate MFA secret and QR code for user.
     */
    @Transactional
    public MfaSetupResponse setupMfa(UUID userId) {
        User user = userRepository.findById(userId)
                .orElseThrow(() -> new IllegalArgumentException("User not found"));

        if (user.isMfaEnabled()) {
            throw new IllegalStateException("MFA already enabled for this user");
        }

        // Generate secret
        String secret = secretGenerator.generate();
        user.setMfaSecret(secret);
        userRepository.save(user);

        // Generate QR code data
        QrData qrData = new QrData.Builder()
                .label(user.getEmail())
                .secret(secret)
                .issuer(issuer)
                .algorithm(HashingAlgorithm.SHA1)
                .digits(6)
                .period(30)
                .build();

        try {
            String qrCodeImage = qrGenerator.generate(qrData);
            return new MfaSetupResponse(secret, qrData.getUri(), qrCodeImage);
        } catch (QrGenerationException e) {
            log.error("Failed to generate QR code", e);
            throw new MfaException("Failed to generate QR code");
        }
    }

    /**
     * Verify and enable MFA for user.
     */
    @Transactional
    public void enableMfa(UUID userId, String code) {
        User user = userRepository.findById(userId)
                .orElseThrow(() -> new IllegalArgumentException("User not found"));

        if (user.isMfaEnabled()) {
            throw new IllegalStateException("MFA already enabled");
        }

        if (!verifyCode(user.getMfaSecret(), code)) {
            throw new InvalidMfaCodeException("Invalid MFA code");
        }

        user.setMfaEnabled(true);
        userRepository.save(user);
        log.info("MFA enabled for user: {}", userId);
    }

    /**
     * Disable MFA for user.
     */
    @Transactional
    public void disableMfa(UUID userId, String code) {
        User user = userRepository.findById(userId)
                .orElseThrow(() -> new IllegalArgumentException("User not found"));

        if (!user.isMfaEnabled()) {
            throw new IllegalStateException("MFA not enabled");
        }

        if (!verifyCode(user.getMfaSecret(), code)) {
            throw new InvalidMfaCodeException("Invalid MFA code");
        }

        user.setMfaEnabled(false);
        user.setMfaSecret(null);
        userRepository.save(user);
        log.info("MFA disabled for user: {}", userId);
    }

    /**
     * Verify MFA code during login.
     */
    public boolean verifyMfaCode(UUID userId, String code) {
        User user = userRepository.findById(userId)
                .orElseThrow(() -> new IllegalArgumentException("User not found"));

        if (!user.isMfaEnabled()) {
            return true; // MFA not required
        }

        return verifyCode(user.getMfaSecret(), code);
    }

    /**
     * Verify TOTP code.
     */
    private boolean verifyCode(String secret, String code) {
        CodeVerifier verifier = new DefaultCodeVerifier(
                new CodeGenerator(),
                timeProvider
        );
        verifier.setTimePeriod(30);
        verifier.setTimeWindow(timeWindow);

        return verifier.isValidCode(secret, code);
    }

    /**
     * Generate current TOTP code (for testing).
     */
    public String generateCurrentCode(String secret) {
        CodeGenerator generator = new CodeGenerator();
        long time = timeProvider.getTime();
        return generator.generateCode(secret, time);
    }

    // Response and exception classes
    public record MfaSetupResponse(String secret, String uri, String qrCodeImage) {}

    public static class MfaException extends RuntimeException {
        public MfaException(String message) {
            super(message);
        }
    }

    public static class InvalidMfaCodeException extends RuntimeException {
        public InvalidMfaCodeException(String message) {
            super(message);
        }
    }
}
