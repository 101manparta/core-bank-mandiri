package com.corebank.auth.dto;

import jakarta.validation.constraints.Email;
import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.Pattern;
import lombok.*;

/**
 * Data Transfer Objects for Auth Service.
 */
public class AuthDto {

    @lombok.Data
    @lombok.NoArgsConstructor
    @lombok.AllArgsConstructor
    public static class LoginRequest {
        @NotBlank(message = "Email is required")
        @Email(message = "Invalid email format")
        private String email;

        @NotBlank(message = "Password is required")
        private String password;

        private String mfaCode;
        private String deviceId;
    }

    @lombok.Data
    @lombok.NoArgsConstructor
    @lombok.AllArgsConstructor
    @lombok.Builder
    public static class AuthResponse {
        private boolean success;
        private boolean mfaRequired;
        private String userId;
        private String email;
        private String accessToken;
        private String refreshToken;
        private String sessionId;
        private String mfaChallenge;

        public static AuthResponse success(UUID userId, String email, String accessToken, 
                                           String refreshToken, String sessionId) {
            AuthResponse response = new AuthResponse();
            response.success = true;
            response.mfaRequired = false;
            response.userId = userId.toString();
            response.email = email;
            response.accessToken = accessToken;
            response.refreshToken = refreshToken;
            response.sessionId = sessionId;
            return response;
        }

        public static AuthResponse mfaRequired(UUID userId) {
            AuthResponse response = new AuthResponse();
            response.success = false;
            response.mfaRequired = true;
            response.userId = userId.toString();
            response.mfaChallenge = "MFA code required";
            return response;
        }
    }

    @lombok.Data
    @lombok.NoArgsConstructor
    @lombok.AllArgsConstructor
    public static class TokenPair {
        private String accessToken;
        private String refreshToken;
    }

    @lombok.Data
    @lombok.NoArgsConstructor
    @lombok.AllArgsConstructor
    public static class RegisterRequest {
        @NotBlank(message = "Email is required")
        @Email(message = "Invalid email format")
        private String email;

        @NotBlank(message = "Password is required")
        @Pattern(
            regexp = "^(?=.*[a-z])(?=.*[A-Z])(?=.*\\d)(?=.*[@$!%*?&])[A-Za-z\\d@$!%*?&]{8,}$",
            message = "Password must be at least 8 characters with uppercase, lowercase, digit, and special character"
        )
        private String password;

        private String phoneNumber;
    }

    @lombok.Data
    @lombok.NoArgsConstructor
    @lombok.AllArgsConstructor
    public static class RefreshTokenRequest {
        @NotBlank(message = "Refresh token is required")
        private String refreshToken;
    }

    @lombok.Data
    @lombok.NoArgsConstructor
    @lombok.AllArgsConstructor
    public static class ChangePasswordRequest {
        @NotBlank(message = "Current password is required")
        private String currentPassword;

        @NotBlank(message = "New password is required")
        @Pattern(
            regexp = "^(?=.*[a-z])(?=.*[A-Z])(?=.*\\d)(?=.*[@$!%*?&])[A-Za-z\\d@$!%*?&]{8,}$",
            message = "Password must be at least 8 characters with uppercase, lowercase, digit, and special character"
        )
        private String newPassword;
    }

    @lombok.Data
    @lombok.NoArgsConstructor
    @lombok.AllArgsConstructor
    public static class MfaSetupRequest {
        private boolean enabled;
    }

    @lombok.Data
    @lombok.NoArgsConstructor
    @lombok.AllArgsConstructor
    public static class MfaVerifyRequest {
        @NotBlank(message = "MFA code is required")
        private String code;
    }

    @lombok.Data
    @lombok.NoArgsConstructor
    @lombok.AllArgsConstructor
    @lombok.Builder
    public static class ApiResponse<T> {
        private boolean success;
        private T data;
        private ErrorDetail error;

        public static <T> ApiResponse<T> success(T data) {
            return ApiResponse.<T>builder()
                    .success(true)
                    .data(data)
                    .build();
        }

        public static <T> ApiResponse<T> error(String code, String message) {
            return ApiResponse.<T>builder()
                    .success(false)
                    .error(new ErrorDetail(code, message))
                    .build();
        }
    }

    @lombok.Data
    @lombok.NoArgsConstructor
    @lombok.AllArgsConstructor
    @lombok.Builder
    public static class ErrorDetail {
        private String code;
        private String message;
    }
}
