-- ==========================================================================
-- Migration: 003_access_grants.sql
-- Description: Create access_grants table for temporary read-only access
-- ==========================================================================

CREATE TABLE IF NOT EXISTS access_grants (
    id SERIAL PRIMARY KEY,
    owner_user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    viewer_user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_word_verified BOOLEAN NOT NULL DEFAULT FALSE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_access_grants_lookup ON access_grants(owner_user_id, viewer_user_id);
