-- Migration: 004_add_note_context.sql
-- Description: Add context column to notes table
ALTER TABLE notes ADD COLUMN IF NOT EXISTS context TEXT NOT NULL DEFAULT '';
