-- Core Bank Mandiri - PostgreSQL Schema
-- Version: 1.0.0
-- Description: Production-grade banking database schema with ACID compliance

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================================
-- ENUM TYPES
-- ============================================================================

CREATE TYPE user_status AS ENUM ('PENDING', 'ACTIVE', 'SUSPENDED', 'CLOSED');
CREATE TYPE account_status AS ENUM ('PENDING', 'ACTIVE', 'FROZEN', 'CLOSED');
CREATE TYPE account_type AS ENUM ('SAVINGS', 'CHECKING', 'BUSINESS', 'INVESTMENT');
CREATE TYPE currency AS ENUM ('IDR', 'USD', 'EUR', 'SGD');
CREATE TYPE transaction_type AS ENUM ('TRANSFER', 'PAYMENT', 'DEPOSIT', 'WITHDRAWAL', 'FEE', 'INTEREST', 'REFUND');
CREATE TYPE transaction_status AS ENUM ('PENDING', 'PROCESSING', 'COMPLETED', 'FAILED', 'REVERSED');
CREATE TYPE ledger_direction AS ENUM ('DEBIT', 'CREDIT');
CREATE TYPE holder_type AS ENUM ('INDIVIDUAL', 'CORPORATE', 'JOINT');
CREATE TYPE fraud_status AS ENUM ('PENDING', 'INVESTIGATING', 'CONFIRMED', 'FALSE_POSITIVE');
CREATE TYPE notification_type AS ENUM ('EMAIL', 'SMS', 'PUSH', 'IN_APP');
CREATE TYPE notification_status AS ENUM ('PENDING', 'SENT', 'DELIVERED', 'FAILED');
CREATE TYPE audit_action AS ENUM ('CREATE', 'READ', 'UPDATE', 'DELETE', 'LOGIN', 'LOGOUT', 'TRANSFER', 'APPROVE', 'REJECT');

-- ============================================================================
-- USERS TABLE
-- ============================================================================

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) NOT NULL UNIQUE,
    phone_number VARCHAR(20),
    password_hash VARCHAR(255) NOT NULL,
    mfa_secret VARCHAR(255),
    mfa_enabled BOOLEAN DEFAULT FALSE,
    status user_status DEFAULT 'PENDING',
    email_verified BOOLEAN DEFAULT FALSE,
    phone_verified BOOLEAN DEFAULT FALSE,
    last_login_at TIMESTAMPTZ,
    failed_login_attempts INTEGER DEFAULT 0,
    locked_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,
    
    CONSTRAINT chk_email_format CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$'),
    CONSTRAINT chk_failed_login_attempts CHECK (failed_login_attempts >= 0)
);

CREATE INDEX idx_users_email ON users(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_phone ON users(phone_number) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_status ON users(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_created_at ON users(created_at);

-- ============================================================================
-- USER PROFILES TABLE
-- ============================================================================

CREATE TABLE user_profiles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    middle_name VARCHAR(100),
    date_of_birth DATE,
    nationality VARCHAR(50),
    id_type VARCHAR(50) NOT NULL, -- KTP, PASSPORT, etc.
    id_number VARCHAR(100) NOT NULL,
    id_expiry_date DATE,
    tax_id VARCHAR(50),
    occupation VARCHAR(100),
    employer VARCHAR(200),
    annual_income NUMERIC(18, 2),
    risk_profile VARCHAR(20) DEFAULT 'MEDIUM', -- LOW, MEDIUM, HIGH
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT uk_user_profiles_user_id UNIQUE (user_id),
    CONSTRAINT uk_user_profiles_id_number UNIQUE (id_type, id_number)
);

CREATE INDEX idx_user_profiles_user_id ON user_profiles(user_id);

-- ============================================================================
-- ADDRESSES TABLE
-- ============================================================================

CREATE TABLE addresses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    address_type VARCHAR(20) NOT NULL, -- HOME, WORK, MAILING
    address_line1 VARCHAR(255) NOT NULL,
    address_line2 VARCHAR(255),
    city VARCHAR(100) NOT NULL,
    state_province VARCHAR(100),
    postal_code VARCHAR(20) NOT NULL,
    country VARCHAR(50) NOT NULL DEFAULT 'Indonesia',
    is_primary BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT chk_address_type CHECK (address_type IN ('HOME', 'WORK', 'MAILING'))
);

CREATE INDEX idx_addresses_user_id ON addresses(user_id);
CREATE INDEX idx_addresses_primary ON addresses(user_id, is_primary) WHERE is_primary = TRUE;

-- ============================================================================
-- ACCOUNTS TABLE
-- ============================================================================

CREATE TABLE accounts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    account_no VARCHAR(20) NOT NULL UNIQUE,
    user_id UUID NOT NULL REFERENCES users(id),
    account_type account_type NOT NULL,
    status account_status DEFAULT 'PENDING',
    balance NUMERIC(18, 2) DEFAULT 0.00 CHECK (balance >= 0),
    available_balance NUMERIC(18, 2) DEFAULT 0.00 CHECK (available_balance >= 0),
    currency currency DEFAULT 'IDR',
    interest_rate NUMERIC(5, 4) DEFAULT 0.0000,
    overdraft_limit NUMERIC(18, 2) DEFAULT 0.00 CHECK (overdraft_limit >= 0),
    minimum_balance NUMERIC(18, 2) DEFAULT 0.00,
    opened_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    closed_at TIMESTAMPTZ,
    closed_reason VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,
    
    CONSTRAINT chk_account_no_format CHECK (account_no ~ '^[0-9]{10,20}$'),
    CONSTRAINT chk_balance_consistency CHECK (available_balance <= balance + overdraft_limit)
);

CREATE INDEX idx_accounts_account_no ON accounts(account_no) WHERE deleted_at IS NULL;
CREATE INDEX idx_accounts_user_id ON accounts(user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_accounts_status ON accounts(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_accounts_type ON accounts(account_type) WHERE deleted_at IS NULL;

-- ============================================================================
-- ACCOUNT HOLDERS TABLE (for joint accounts)
-- ============================================================================

CREATE TABLE account_holders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    holder_type holder_type NOT NULL,
    holder_name VARCHAR(255) NOT NULL,
    percentage NUMERIC(5, 2) DEFAULT 100.00 CHECK (percentage > 0 AND percentage <= 100),
    is_primary BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT uk_account_holders_account_user UNIQUE (account_id, user_id),
    CONSTRAINT chk_holder_type CHECK (holder_type IN ('INDIVIDUAL', 'CORPORATE', 'JOINT'))
);

CREATE INDEX idx_account_holders_account_id ON account_holders(account_id);
CREATE INDEX idx_account_holders_user_id ON account_holders(user_id);

-- ============================================================================
-- ACCOUNT LIMITS TABLE
-- ============================================================================

CREATE TABLE account_limits (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    daily_transfer_limit NUMERIC(18, 2) DEFAULT 10000000.00,
    daily_withdrawal_limit NUMERIC(18, 2) DEFAULT 5000000.00,
    single_transaction_limit NUMERIC(18, 2) DEFAULT 5000000.00,
    monthly_transfer_limit NUMERIC(18, 2) DEFAULT 100000000.00,
    daily_transfer_used NUMERIC(18, 2) DEFAULT 0.00,
    daily_withdrawal_used NUMERIC(18, 2) DEFAULT 0.00,
    monthly_transfer_used NUMERIC(18, 2) DEFAULT 0.00,
    limit_reset_date DATE DEFAULT CURRENT_DATE,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT uk_account_limits_account_id UNIQUE (account_id)
);

CREATE INDEX idx_account_limits_account_id ON account_limits(account_id);

-- ============================================================================
-- TRANSACTIONS TABLE
-- ============================================================================

CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    reference VARCHAR(50) NOT NULL UNIQUE,
    idempotency_key VARCHAR(100) UNIQUE,
    transaction_type transaction_type NOT NULL,
    status transaction_status DEFAULT 'PENDING',
    amount NUMERIC(18, 2) NOT NULL CHECK (amount > 0),
    fee_amount NUMERIC(18, 2) DEFAULT 0.00 CHECK (fee_amount >= 0),
    total_amount NUMERIC(18, 2) NOT NULL CHECK (total_amount > 0),
    currency currency DEFAULT 'IDR',
    from_account_id UUID REFERENCES accounts(id),
    to_account_id UUID REFERENCES accounts(id),
    to_account_number VARCHAR(20),
    to_bank_code VARCHAR(10),
    to_account_name VARCHAR(255),
    description TEXT,
    metadata JSONB DEFAULT '{}',
    failure_reason VARCHAR(255),
    reversal_transaction_id UUID REFERENCES transactions(id),
    processed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT chk_total_amount CHECK (total_amount = amount + fee_amount),
    CONSTRAINT chk_transaction_direction CHECK (
        (transaction_type IN ('TRANSFER', 'PAYMENT') AND from_account_id IS NOT NULL) OR
        (transaction_type IN ('DEPOSIT') AND to_account_id IS NOT NULL) OR
        (transaction_type IN ('WITHDRAWAL') AND from_account_id IS NOT NULL)
    )
);

CREATE INDEX idx_transactions_reference ON transactions(reference) WHERE status != 'FAILED';
CREATE INDEX idx_transactions_from_account ON transactions(from_account_id) WHERE created_at > CURRENT_TIMESTAMP - INTERVAL '90 days';
CREATE INDEX idx_transactions_to_account ON transactions(to_account_id) WHERE created_at > CURRENT_TIMESTAMP - INTERVAL '90 days';
CREATE INDEX idx_transactions_status ON transactions(status);
CREATE INDEX idx_transactions_type ON transactions(transaction_type);
CREATE INDEX idx_transactions_created_at ON transactions(created_at);
CREATE INDEX idx_transactions_idempotency ON transactions(idempotency_key) WHERE idempotency_key IS NOT NULL;

-- ============================================================================
-- LEDGER ENTRIES TABLE (Double-Entry Accounting)
-- ============================================================================

CREATE TABLE ledger_entries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    transaction_id UUID NOT NULL REFERENCES transactions(id),
    entry_number INTEGER NOT NULL,
    account_id UUID NOT NULL REFERENCES accounts(id),
    direction ledger_direction NOT NULL,
    amount NUMERIC(18, 2) NOT NULL CHECK (amount > 0),
    balance_before NUMERIC(18, 2) NOT NULL,
    balance_after NUMERIC(18, 2) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT uk_transaction_entry UNIQUE (transaction_id, entry_number),
    CONSTRAINT chk_balance_movement CHECK (
        (direction = 'DEBIT' AND balance_after = balance_before - amount) OR
        (direction = 'CREDIT' AND balance_after = balance_before + amount)
    )
);

CREATE INDEX idx_ledger_entries_transaction_id ON ledger_entries(transaction_id);
CREATE INDEX idx_ledger_entries_account_id ON ledger_entries(account_id, created_at);
CREATE INDEX idx_ledger_entries_created_at ON ledger_entries(created_at);

-- Prevent updates and deletes on ledger entries (immutability)
CREATE OR REPLACE FUNCTION prevent_ledger_modification()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'UPDATE' THEN
        IF OLD.amount != NEW.amount OR OLD.direction != NEW.direction OR OLD.account_id != NEW.account_id THEN
            RAISE EXCEPTION 'Ledger entries are immutable. Cannot modify critical fields.';
        END IF;
    ELSIF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'Ledger entries cannot be deleted.';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_ledger_entries_immutable
    BEFORE UPDATE OR DELETE ON ledger_entries
    FOR EACH ROW EXECUTE FUNCTION prevent_ledger_modification();

-- ============================================================================
-- SESSIONS TABLE
-- ============================================================================

CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    refresh_token_hash VARCHAR(255),
    device_id VARCHAR(255),
    device_info JSONB,
    ip_address INET,
    user_agent TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    refreshed_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT chk_session_expiry CHECK (expires_at > created_at)
);

CREATE INDEX idx_sessions_user_id ON sessions(user_id) WHERE revoked_at IS NULL;
CREATE INDEX idx_sessions_token_hash ON sessions(token_hash);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at) WHERE revoked_at IS NULL;

-- ============================================================================
-- OTP TABLE (for MFA and verification)
-- ============================================================================

CREATE TABLE otp_codes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    otp_type VARCHAR(20) NOT NULL, -- LOGIN, TRANSACTION, PASSWORD_RESET, EMAIL_VERIFY
    code_hash VARCHAR(255) NOT NULL,
    channel VARCHAR(20) NOT NULL, -- SMS, EMAIL, TOTP
    attempts INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 3,
    verified BOOLEAN DEFAULT FALSE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT chk_otp_type CHECK (otp_type IN ('LOGIN', 'TRANSACTION', 'PASSWORD_RESET', 'EMAIL_VERIFY', 'PHONE_VERIFY')),
    CONSTRAINT chk_otp_attempts CHECK (attempts >= 0 AND attempts <= max_attempts)
);

CREATE INDEX idx_otp_codes_user_id ON otp_codes(user_id, otp_type) WHERE NOT verified AND expires_at > CURRENT_TIMESTAMP;
CREATE INDEX idx_otp_codes_expires_at ON otp_codes(expires_at);

-- ============================================================================
-- FRAUD ALERTS TABLE
-- ============================================================================

CREATE TABLE fraud_alerts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    transaction_id UUID REFERENCES transactions(id),
    user_id UUID NOT NULL REFERENCES users(id),
    alert_type VARCHAR(50) NOT NULL,
    risk_score INTEGER NOT NULL CHECK (risk_score >= 0 AND risk_score <= 100),
    status fraud_status DEFAULT 'PENDING',
    reason TEXT,
    rules_triggered JSONB,
    reviewed_by UUID REFERENCES users(id),
    reviewed_at TIMESTAMPTZ,
    resolution_notes TEXT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT chk_risk_score CHECK (risk_score >= 0 AND risk_score <= 100)
);

CREATE INDEX idx_fraud_alerts_transaction_id ON fraud_alerts(transaction_id);
CREATE INDEX idx_fraud_alerts_user_id ON fraud_alerts(user_id);
CREATE INDEX idx_fraud_alerts_status ON fraud_alerts(status);
CREATE INDEX idx_fraud_alerts_risk_score ON fraud_alerts(risk_score) WHERE status = 'PENDING';
CREATE INDEX idx_fraud_alerts_created_at ON fraud_alerts(created_at);

-- ============================================================================
-- AUDIT LOGS TABLE
-- ============================================================================

CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    trace_id VARCHAR(100),
    user_id UUID REFERENCES users(id),
    action audit_action NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id VARCHAR(100),
    old_values JSONB,
    new_values JSONB,
    ip_address INET,
    user_agent TEXT,
    service_name VARCHAR(100),
    status VARCHAR(20) DEFAULT 'SUCCESS', -- SUCCESS, FAILURE
    error_message TEXT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);
CREATE INDEX idx_audit_logs_trace_id ON audit_logs(trace_id);

-- Partition audit_logs by date for performance
-- CREATE TABLE audit_logs_y2024m01 PARTITION OF audit_logs
--     FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

-- ============================================================================
-- NOTIFICATIONS TABLE
-- ============================================================================

CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    notification_type notification_type NOT NULL,
    status notification_status DEFAULT 'PENDING',
    subject VARCHAR(255),
    body TEXT NOT NULL,
    template_name VARCHAR(100),
    template_data JSONB,
    channel_data JSONB,
    sent_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    read_at TIMESTAMPTZ,
    failed_reason VARCHAR(255),
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT chk_notification_type CHECK (notification_type IN ('EMAIL', 'SMS', 'PUSH', 'IN_APP')),
    CONSTRAINT chk_retry_count CHECK (retry_count >= 0 AND retry_count <= max_retries)
);

CREATE INDEX idx_notifications_user_id ON notifications(user_id) WHERE status != 'SENT';
CREATE INDEX idx_notifications_status ON notifications(status);
CREATE INDEX idx_notifications_type ON notifications(notification_type);
CREATE INDEX idx_notifications_created_at ON notifications(created_at);

-- ============================================================================
-- NOTIFICATION PREFERENCES TABLE
-- ============================================================================

CREATE TABLE notification_preferences (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    notification_type notification_type NOT NULL,
    event_category VARCHAR(50) NOT NULL, -- TRANSACTION, SECURITY, MARKETING, SYSTEM
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT uk_user_notification_event UNIQUE (user_id, notification_type, event_category)
);

CREATE INDEX idx_notification_preferences_user_id ON notification_preferences(user_id);

-- ============================================================================
-- EXCHANGE RATES TABLE
-- ============================================================================

CREATE TABLE exchange_rates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    from_currency currency NOT NULL,
    to_currency currency NOT NULL,
    rate NUMERIC(18, 8) NOT NULL CHECK (rate > 0),
    inverse_rate NUMERIC(18, 8) NOT NULL CHECK (inverse_rate > 0),
    effective_from TIMESTAMPTZ NOT NULL,
    effective_until TIMESTAMPTZ,
    source VARCHAR(50),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT uk_currency_pair_date UNIQUE (from_currency, to_currency, effective_from)
);

CREATE INDEX idx_exchange_rates_currencies ON exchange_rates(from_currency, to_currency, effective_from);
CREATE INDEX idx_exchange_rates_effective ON exchange_rates(effective_from, effective_until);

-- ============================================================================
-- SYSTEM CONFIGURATION TABLE
-- ============================================================================

CREATE TABLE system_config (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    config_key VARCHAR(100) NOT NULL UNIQUE,
    config_value JSONB NOT NULL,
    description TEXT,
    is_sensitive BOOLEAN DEFAULT FALSE,
    updated_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_system_config_key ON system_config(config_key);

-- ============================================================================
-- API KEYS TABLE (for service-to-service auth)
-- ============================================================================

CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    key_hash VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    service_name VARCHAR(100),
    permissions JSONB NOT NULL,
    rate_limit INTEGER DEFAULT 1000, -- requests per minute
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT chk_api_key_expiry CHECK (expires_at IS NULL OR expires_at > created_at)
);

CREATE INDEX idx_api_keys_key_hash ON api_keys(key_hash) WHERE revoked_at IS NULL;
CREATE INDEX idx_api_keys_service_name ON api_keys(service_name) WHERE revoked_at IS NULL;

-- ============================================================================
-- RATE LIMIT TRACKING TABLE
-- ============================================================================

CREATE TABLE rate_limit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    identifier VARCHAR(255) NOT NULL, -- IP, user_id, api_key
    endpoint VARCHAR(255) NOT NULL,
    request_count INTEGER DEFAULT 1,
    window_start TIMESTAMPTZ NOT NULL,
    window_end TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT uk_rate_limit_identifier_window UNIQUE (identifier, endpoint, window_start)
);

CREATE INDEX idx_rate_limit_logs_identifier ON rate_limit_logs(identifier, window_end);
CREATE INDEX idx_rate_limit_logs_window ON rate_limit_logs(window_end);

-- ============================================================================
-- SCHEDULED TRANSACTIONS TABLE
-- ============================================================================

CREATE TABLE scheduled_transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id),
    from_account_id UUID NOT NULL REFERENCES accounts(id),
    to_account_id UUID REFERENCES accounts(id),
    to_account_number VARCHAR(20),
    to_bank_code VARCHAR(10),
    amount NUMERIC(18, 2) NOT NULL CHECK (amount > 0),
    description VARCHAR(255),
    schedule_type VARCHAR(20) NOT NULL, -- ONCE, DAILY, WEEKLY, MONTHLY
    schedule_config JSONB, -- {day_of_week: 1, day_of_month: 15, etc.}
    next_execution_at TIMESTAMPTZ NOT NULL,
    last_execution_at TIMESTAMPTZ,
    status VARCHAR(20) DEFAULT 'ACTIVE', -- ACTIVE, PAUSED, COMPLETED, CANCELLED
    execution_count INTEGER DEFAULT 0,
    max_executions INTEGER,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT chk_schedule_type CHECK (schedule_type IN ('ONCE', 'DAILY', 'WEEKLY', 'MONTHLY'))
);

CREATE INDEX idx_scheduled_transactions_user_id ON scheduled_transactions(user_id);
CREATE INDEX idx_scheduled_transactions_next_exec ON scheduled_transactions(next_execution_at) WHERE status = 'ACTIVE';

-- ============================================================================
-- BENEFICIARIES TABLE
-- ============================================================================

CREATE TABLE beneficiaries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    account_number VARCHAR(20) NOT NULL,
    bank_code VARCHAR(10),
    bank_name VARCHAR(100),
    account_type VARCHAR(50),
    is_verified BOOLEAN DEFAULT FALSE,
    verification_method VARCHAR(50),
    nickname VARCHAR(100),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT uk_beneficiaries_user_account UNIQUE (user_id, account_number, bank_code)
);

CREATE INDEX idx_beneficiaries_user_id ON beneficiaries(user_id);

-- ============================================================================
-- TRIGGERS FOR UPDATED_AT
-- ============================================================================

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply to tables with updated_at
CREATE TRIGGER trg_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_user_profiles_updated_at BEFORE UPDATE ON user_profiles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_addresses_updated_at BEFORE UPDATE ON addresses
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_accounts_updated_at BEFORE UPDATE ON accounts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_account_limits_updated_at BEFORE UPDATE ON account_limits
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_transactions_updated_at BEFORE UPDATE ON transactions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_system_config_updated_at BEFORE UPDATE ON system_config
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_notification_preferences_updated_at BEFORE UPDATE ON notification_preferences
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_beneficiaries_updated_at BEFORE UPDATE ON beneficiaries
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- TRIGGER FOR DAILY LIMIT RESET
-- ============================================================================

CREATE OR REPLACE FUNCTION reset_daily_limits()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.limit_reset_date != OLD.limit_reset_date THEN
        NEW.daily_transfer_used := 0;
        NEW.daily_withdrawal_used := 0;
        NEW.limit_reset_date := CURRENT_DATE;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_account_limits_reset
    BEFORE UPDATE ON account_limits
    FOR EACH ROW EXECUTE FUNCTION reset_daily_limits();

-- ============================================================================
-- FUNCTION: Create Ledger Entry (Double-Entry)
-- ============================================================================

CREATE OR REPLACE FUNCTION create_double_entry_transaction(
    p_transaction_id UUID,
    p_from_account_id UUID,
    p_to_account_id UUID,
    p_amount NUMERIC,
    p_fee NUMERIC DEFAULT 0,
    p_description TEXT DEFAULT NULL
) RETURNS VOID AS $$
DECLARE
    v_from_balance NUMERIC;
    v_to_balance NUMERIC;
    v_total_amount NUMERIC;
BEGIN
    v_total_amount := p_amount + p_fee;
    
    -- Lock accounts for update
    SELECT balance INTO v_from_balance FROM accounts WHERE id = p_from_account_id FOR UPDATE;
    SELECT balance INTO v_to_balance FROM accounts WHERE id = p_to_account_id FOR UPDATE;
    
    -- Validate sufficient balance
    IF v_from_balance < v_total_amount THEN
        RAISE EXCEPTION 'Insufficient balance. Required: %, Available: %', v_total_amount, v_from_balance;
    END IF;
    
    -- Create debit entry (from account)
    INSERT INTO ledger_entries (transaction_id, entry_number, account_id, direction, amount, balance_before, balance_after, description)
    VALUES (p_transaction_id, 1, p_from_account_id, 'DEBIT', v_total_amount, v_from_balance, v_from_balance - v_total_amount, p_description);
    
    -- Create credit entry (to account)
    INSERT INTO ledger_entries (transaction_id, entry_number, account_id, direction, amount, balance_before, balance_after, description)
    VALUES (p_transaction_id, 2, p_to_account_id, 'CREDIT', p_amount, v_to_balance, v_to_balance + p_amount, p_description);
    
    -- If there's a fee, create fee credit entry
    IF p_fee > 0 THEN
        -- Fee goes to bank revenue account (would need a bank internal account)
        INSERT INTO ledger_entries (transaction_id, entry_number, account_id, direction, amount, balance_before, balance_after, description)
        VALUES (p_transaction_id, 3, p_from_account_id, 'DEBIT', p_fee, 0, p_fee, 'Transaction fee');
    END IF;
    
    -- Update account balances
    UPDATE accounts SET 
        balance = balance - v_total_amount,
        available_balance = available_balance - v_total_amount,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = p_from_account_id;
    
    UPDATE accounts SET 
        balance = balance + p_amount,
        available_balance = available_balance + p_amount,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = p_to_account_id;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- FUNCTION: Generate Account Number
-- ============================================================================

CREATE OR REPLACE FUNCTION generate_account_number(p_account_type account_type)
RETURNS VARCHAR(20) AS $$
DECLARE
    v_prefix VARCHAR(4);
    v_sequence BIGINT;
    v_check_digit INTEGER;
    v_account_no VARCHAR(20);
BEGIN
    -- Set prefix based on account type
    CASE p_account_type
        WHEN 'SAVINGS' THEN v_prefix := '10';
        WHEN 'CHECKING' THEN v_prefix := '20';
        WHEN 'BUSINESS' THEN v_prefix := '30';
        WHEN 'INVESTMENT' THEN v_prefix := '40';
    END CASE;
    
    -- Get next sequence value
    SELECT COALESCE(MAX(SUBSTRING(account_no FROM 5 FOR 12)::BIGINT), 0) + 1
    INTO v_sequence
    FROM accounts
    WHERE account_no LIKE v_prefix || '%';
    
    -- Generate account number with check digit
    v_account_no := v_prefix || LPAD(v_sequence::TEXT, 12, '0');
    
    -- Calculate check digit (simple modulo 10)
    v_check_digit := 0;
    FOR i IN 1..LENGTH(v_account_no) LOOP
        v_check_digit := v_check_digit + SUBSTRING(v_account_no FROM i FOR 1)::INTEGER;
    END LOOP;
    v_check_digit := (10 - (v_check_digit % 10)) % 10;
    
    RETURN v_account_no || v_check_digit::TEXT;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- SEED DATA
-- ============================================================================

-- Insert system configuration
INSERT INTO system_config (config_key, config_value, description) VALUES
    ('transaction.limits.daily', '{"transfer": 100000000, "withdrawal": 50000000}', 'Daily transaction limits in IDR'),
    ('transaction.limits.single', '{"transfer": 25000000, "withdrawal": 10000000}', 'Single transaction limits in IDR'),
    ('fraud.thresholds', '{"high_risk": 80, "medium_risk": 50, "low_risk": 20}', 'Fraud risk score thresholds'),
    ('session.timeout', '{"access_token": 900, "refresh_token": 604800}', 'Session timeout in seconds'),
    ('otp.expiry', '{"login": 300, "transaction": 600}', 'OTP expiry in seconds');

-- Insert default notification preferences
INSERT INTO notification_preferences (user_id, notification_type, event_category)
SELECT id, 'EMAIL', 'TRANSACTION' FROM users WHERE FALSE; -- Template for new users

COMMENT ON DATABASE core_bank IS 'Core Bank Mandiri - Production Banking Database';
