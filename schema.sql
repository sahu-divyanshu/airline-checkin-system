-- ============================================================
-- Airline Check-in System — Schema
-- Run once: sudo -u postgres psql -f schema.sql
-- ============================================================

-- Drop and recreate cleanly
DROP DATABASE IF EXISTS airline_db;
CREATE DATABASE airline_db;
\connect airline_db

-- Users table
CREATE TABLE users (
    id   SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL
);

-- Seats table
CREATE TABLE seats (
    id      SERIAL PRIMARY KEY,
    status  VARCHAR(20)  NOT NULL DEFAULT 'available',   -- 'available' | 'booked'
    user_id INTEGER      REFERENCES users(id)            -- NULL until booked
);

-- Insert 5 sample users
INSERT INTO users (name) VALUES
    ('Alice'), ('Bob'), ('Carol'), ('Dave'), ('Eve');

-- ── Step 2 & 3: Single seat (race condition demo) ──────────────────────────
-- Insert exactly ONE seat, status = available, user_id = NULL
INSERT INTO seats (status, user_id) VALUES ('available', NULL);
-- This gives us seat id=1

-- ── Step 4: High-throughput queue (SKIP LOCKED demo) ──────────────────────
-- Add 100 more available seats (ids 2–101)
INSERT INTO seats (status, user_id)
SELECT 'available', NULL FROM generate_series(1, 100);

-- Verify
SELECT 'Total seats:' AS label, COUNT(*) AS count FROM seats;
SELECT 'Available:'   AS label, COUNT(*) AS count FROM seats WHERE status = 'available';
