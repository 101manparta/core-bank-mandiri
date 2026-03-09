# Core Bank Mandiri - Auth Service Dockerfile
# Multi-stage build for optimized production image

# ============================================================================
# Stage 1: Build
# ============================================================================
FROM maven:3.9.6-eclipse-temurin-17 AS builder

WORKDIR /app

# Copy pom.xml and download dependencies (cached layer)
COPY pom.xml .
RUN mvn dependency:go-offline -B

# Copy source code and build
COPY src ./src
RUN mvn clean package -DskipTests -B

# ============================================================================
# Stage 2: Production Runtime
# ============================================================================
FROM eclipse-temurin:17-jre-alpine

# Create non-root user for security
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

# Install curl for health checks
RUN apk --no-cache add curl

# Copy built artifact
COPY --from=builder /app/target/*.jar app.jar

# Create log directory
RUN mkdir -p /var/log/auth-service && \
    chown -R appuser:appgroup /var/log/auth-service && \
    chmod -R 755 /var/log/auth-service

# Set ownership
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# JVM options for production
ENV JAVA_OPTS="-Xms256m -Xmx512m \
    -XX:+UseG1GC \
    -XX:MaxGCPauseMillis=200 \
    -XX:+HeapDumpOnOutOfMemoryError \
    -XX:HeapDumpPath=/var/log/auth-service \
    -Djava.security.egd=file:/dev/./urandom \
    -Dserver.port=8080"

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD curl -f http://localhost:8080/actuator/health || exit 1

# Run application
ENTRYPOINT ["sh", "-c", "java $JAVA_OPTS -jar app.jar"]
