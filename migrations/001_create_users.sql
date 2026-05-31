-- ==========================================================================
-- Migration: 001_create_users.sql
-- Description: Create users table for engineer credentials in Cozy Cluster
-- ==========================================================================

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(100) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    code_word1 VARCHAR(100),
    code_word2 VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Index for high-performance authentication lookups
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
