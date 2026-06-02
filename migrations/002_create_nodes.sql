-- ==========================================================================
-- Migration: 002_create_nodes.sql
-- Description: Create notes and connections tables for Cozy Canvas layout
-- ==========================================================================

-- Notes table: holds coordinates, type (is_env), and markdown content of nodes
CREATE TABLE IF NOT EXISTS notes (
    id VARCHAR(50) PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE, -- NULL means global env note
    text TEXT NOT NULL,
    x DOUBLE PRECISION NOT NULL,
    y DOUBLE PRECISION NOT NULL,
    is_env BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_notes_user_id ON notes(user_id);
CREATE INDEX IF NOT EXISTS idx_notes_is_env ON notes(is_env);

-- Connections table: links notes together using force physics spring lines
CREATE TABLE IF NOT EXISTS connections (
    id VARCHAR(100) PRIMARY KEY, -- source_id-target_id
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    source_note_id VARCHAR(50) NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    target_note_id VARCHAR(50) NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_connections_user_id ON connections(user_id);
