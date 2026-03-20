-- ============================================================
-- 043_phase2_task1_mapping_primary_key.sql
-- Switch coin_exchange_mappings to the identity `id` primary key
-- so one coin can have multiple exchange symbols.
-- ============================================================

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conrelid = 'coin_exchange_mappings'::regclass
          AND conname = 'coin_exchange_mappings_pkey'
    ) THEN
        ALTER TABLE coin_exchange_mappings
            DROP CONSTRAINT coin_exchange_mappings_pkey;
    END IF;
END $$;

ALTER TABLE coin_exchange_mappings
    ADD PRIMARY KEY (id);

SELECT setval(
    pg_get_serial_sequence('coin_exchange_mappings', 'id'),
    COALESCE((SELECT MAX(id) FROM coin_exchange_mappings), 1),
    TRUE
);
