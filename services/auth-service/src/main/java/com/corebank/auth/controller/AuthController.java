package com.corebank.auth.controller;

import com.corebank.auth.dto.AuthDto;
import com.corebank.auth.service.AuthenticationService;
import com.corebank.auth.service.MfaService;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.validation.Valid;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.http.ResponseEntity;
import org.springframework.security.access.prepost.PreAuthorize;
import org.springframework.web.bind.annotation.*;

import java.util.Map;
import java.util.UUID;

/**
 * REST Controller for authentication endpoints.
 */
@Slf4j
@RestController
@RequestMapping("/api/v1/auth")
@RequiredArgsConstructor
public class AuthController {

    private final AuthenticationService authenticationService;
    private final MfaService mfaService;

    /**
     * User login endpoint.
     */
    @PostMapping("/login")
    public ResponseEntity<AuthDto.ApiResponse<AuthDto.AuthResponse>> login(
            @Valid @RequestBody AuthDto.LoginRequest request,
            HttpServletRequest httpRequest) {
        
        log.info("Login request received for email: {}", request.getEmail());
        
        String ipAddress = getClientIp(httpRequest);
        String userAgent = httpRequest.getHeader("User-Agent");
        
        AuthDto.AuthResponse response = authenticationService.login(request, ipAddress, userAgent);
        
        return ResponseEntity.ok(AuthDto.ApiResponse.success(response));
    }

    /**
     * User logout endpoint.
     */
    @PostMapping("/logout")
    @PreAuthorize("isAuthenticated()")
    public ResponseEntity<AuthDto.ApiResponse<Void>> logout(
            @RequestHeader("Authorization") String authorization) {
        
        String token = extractToken(authorization);
        authenticationService.logout(token);
        
        return ResponseEntity.ok(AuthDto.ApiResponse.success(null));
    }

    /**
     * Refresh access token endpoint.
     */
    @PostMapping("/refresh")
    public ResponseEntity<AuthDto.ApiResponse<AuthDto.TokenPair>> refreshToken(
            @Valid @RequestBody AuthDto.RefreshTokenRequest request) {
        
        AuthDto.TokenPair tokens = authenticationService.refreshToken(request.getRefreshToken());
        
        return ResponseEntity.ok(AuthDto.ApiResponse.success(tokens));
    }

    /**
     * User registration endpoint.
     */
    @PostMapping("/register")
    public ResponseEntity<AuthDto.ApiResponse<Map<String, String>>> register(
            @Valid @RequestBody AuthDto.RegisterRequest request) {
        
        var user = authenticationService.register(request);
        
        return ResponseEntity.ok(AuthDto.ApiResponse.success(
                Map.of("userId", user.getId().toString(), "email", user.getEmail())
        ));
    }

    /**
     * Change password endpoint.
     */
    @PostMapping("/change-password")
    @PreAuthorize("isAuthenticated()")
    public ResponseEntity<AuthDto.ApiResponse<Void>> changePassword(
            @Valid @RequestBody AuthDto.ChangePasswordRequest request,
            @RequestHeader("X-User-Id") String userIdHeader) {
        
        UUID userId = UUID.fromString(userIdHeader);
        authenticationService.changePassword(userId, request.getCurrentPassword(), request.getNewPassword());
        
        return ResponseEntity.ok(AuthDto.ApiResponse.success(null));
    }

    /**
     * Setup MFA endpoint.
     */
    @PostMapping("/mfa/setup")
    @PreAuthorize("isAuthenticated()")
    public ResponseEntity<AuthDto.ApiResponse<MfaService.MfaSetupResponse>> setupMfa(
            @RequestHeader("X-User-Id") String userIdHeader) {
        
        UUID userId = UUID.fromString(userIdHeader);
        MfaService.MfaSetupResponse response = mfaService.setupMfa(userId);
        
        return ResponseEntity.ok(AuthDto.ApiResponse.success(response));
    }

    /**
     * Enable MFA endpoint.
     */
    @PostMapping("/mfa/enable")
    @PreAuthorize("isAuthenticated()")
    public ResponseEntity<AuthDto.ApiResponse<Void>> enableMfa(
            @Valid @RequestBody AuthDto.MfaVerifyRequest request,
            @RequestHeader("X-User-Id") String userIdHeader) {
        
        UUID userId = UUID.fromString(userIdHeader);
        mfaService.enableMfa(userId, request.getCode());
        
        return ResponseEntity.ok(AuthDto.ApiResponse.success(null));
    }

    /**
     * Disable MFA endpoint.
     */
    @PostMapping("/mfa/disable")
    @PreAuthorize("isAuthenticated()")
    public ResponseEntity<AuthDto.ApiResponse<Void>> disableMfa(
            @Valid @RequestBody AuthDto.MfaVerifyRequest request,
            @RequestHeader("X-User-Id") String userIdHeader) {
        
        UUID userId = UUID.fromString(userIdHeader);
        mfaService.disableMfa(userId, request.getCode());
        
        return ResponseEntity.ok(AuthDto.ApiResponse.success(null));
    }

    /**
     * Verify MFA code endpoint.
     */
    @PostMapping("/mfa/verify")
    public ResponseEntity<AuthDto.ApiResponse<Map<String, Boolean>>> verifyMfa(
            @Valid @RequestBody AuthDto.MfaVerifyRequest request,
            @RequestParam UUID userId) {
        
        boolean valid = mfaService.verifyMfaCode(userId, request.getCode());
        
        return ResponseEntity.ok(AuthDto.ApiResponse.success(
                Map.of("valid", valid)
        ));
    }

    /**
     * Health check endpoint.
     */
    @GetMapping("/health")
    public ResponseEntity<Map<String, String>> health() {
        return ResponseEntity.ok(Map.of("status", "UP", "service", "auth-service"));
    }

    // Helper methods
    private String extractToken(String authorization) {
        if (authorization != null && authorization.startsWith("Bearer ")) {
            return authorization.substring(7);
        }
        return authorization;
    }

    private String getClientIp(HttpServletRequest request) {
        String xForwardedFor = request.getHeader("X-Forwarded-For");
        if (xForwardedFor != null && !xForwardedFor.isEmpty()) {
            return xForwardedFor.split(",")[0].trim();
        }
        
        String xRealIp = request.getHeader("X-Real-IP");
        if (xRealIp != null && !xRealIp.isEmpty()) {
            return xRealIp;
        }
        
        return request.getRemoteAddr();
    }
}
