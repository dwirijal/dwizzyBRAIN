-- ============================================================
-- 001_init_extensions.sql
-- Core PostgreSQL extensions required by dwizzyOS
-- ============================================================

CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;
CREATE EXTENSION IF NOT EXISTS vector;         -- pgvector — embedding similarity search
CREATE EXTENSION IF NOT EXISTS postgis;        -- spatial data (future geo-based signals)
CREATE EXTENSION IF NOT EXISTS pg_trgm;        -- trigram similarity — fuzzy symbol matching
CREATE EXTENSION IF NOT EXISTS btree_gin;      -- GIN index support for composite queries
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";    -- uuid_generate_v4()