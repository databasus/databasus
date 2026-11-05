-- +goose Up
ALTER TABLE verification_agents RENAME COLUMN max_ram_gb TO max_ram_mb;

ALTER TABLE verification_agents ALTER COLUMN max_ram_mb TYPE BIGINT;

UPDATE verification_agents SET max_ram_mb = GREATEST(max_ram_mb, 0) * 1024;

ALTER TABLE verification_agents
    ADD CONSTRAINT ck_verification_agents_max_ram_mb
    CHECK (max_ram_mb >= 0);

-- +goose Down
ALTER TABLE verification_agents DROP CONSTRAINT ck_verification_agents_max_ram_mb;

UPDATE verification_agents SET max_ram_mb = max_ram_mb / 1024;

ALTER TABLE verification_agents ALTER COLUMN max_ram_mb TYPE INTEGER;

ALTER TABLE verification_agents RENAME COLUMN max_ram_mb TO max_ram_gb;
