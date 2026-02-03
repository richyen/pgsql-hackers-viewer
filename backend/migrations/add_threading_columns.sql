-- Migration: Add In-Reply-To and References columns for proper email threading
-- Date: 2026-02-03

-- Add new columns to messages table
ALTER TABLE messages ADD COLUMN IF NOT EXISTS in_reply_to VARCHAR(255) DEFAULT '';
ALTER TABLE messages ADD COLUMN IF NOT EXISTS references TEXT DEFAULT '';

-- Create index for faster threading lookups
CREATE INDEX IF NOT EXISTS idx_messages_in_reply_to ON messages(in_reply_to);

-- Display migration success message
DO $$
BEGIN
    RAISE NOTICE 'Migration completed: Added in_reply_to and references columns to messages table';
END $$;
