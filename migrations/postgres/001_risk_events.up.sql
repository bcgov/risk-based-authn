-- 0001_init_schema.up.sql
CREATE TABLE events (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    score DOUBLE PRECISION,
    details JSONB
);