-- +goose Up
-- +goose StatementBegin
ALTER TABLE email_notifiers
    RENAME TO notifiers_email;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_email
    RENAME CONSTRAINT email_notifiers_pkey TO notifiers_email_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE telegram_notifiers
    RENAME TO notifiers_telegram;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_telegram
    RENAME CONSTRAINT telegram_notifiers_pkey TO notifiers_telegram_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE slack_notifiers
    RENAME TO notifiers_slack;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_slack
    RENAME CONSTRAINT slack_notifiers_pkey TO notifiers_slack_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE discord_notifiers
    RENAME TO notifiers_discord;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_discord
    RENAME CONSTRAINT discord_notifiers_pkey TO notifiers_discord_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE teams_notifiers
    RENAME TO notifiers_teams;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_teams
    RENAME CONSTRAINT teams_notifiers_pkey TO notifiers_teams_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE mattermost_notifiers
    RENAME TO notifiers_mattermost;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_mattermost
    RENAME CONSTRAINT mattermost_notifiers_pkey TO notifiers_mattermost_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE webhook_notifiers
    RENAME TO notifiers_webhook;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_webhook
    RENAME CONSTRAINT webhook_notifiers_pkey TO notifiers_webhook_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_email
    DROP CONSTRAINT ck_email_notifiers_security;
-- +goose StatementEnd

-- A webhook with a method it cannot send keeps sending with the default one.
-- +goose StatementBegin
UPDATE notifiers_webhook
SET webhook_method = 'POST'
WHERE webhook_method NOT IN ('POST', 'GET');
-- +goose StatementEnd

-- A Mattermost row with an unknown delivery mode cannot send; its notifier goes with the broken ones below.
-- +goose StatementBegin
DELETE FROM notifiers_mattermost
WHERE delivery_mode NOT IN ('WEBHOOK', 'BOT');
-- +goose StatementEnd

-- A provider row whose notifier has another type is never read.
-- +goose StatementBegin
DELETE FROM notifiers_email p
USING notifiers n
WHERE n.id = p.notifier_id
    AND n.notifier_type <> 'EMAIL';
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM notifiers_telegram p
USING notifiers n
WHERE n.id = p.notifier_id
    AND n.notifier_type <> 'TELEGRAM';
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM notifiers_slack p
USING notifiers n
WHERE n.id = p.notifier_id
    AND n.notifier_type <> 'SLACK';
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM notifiers_discord p
USING notifiers n
WHERE n.id = p.notifier_id
    AND n.notifier_type <> 'DISCORD';
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM notifiers_teams p
USING notifiers n
WHERE n.id = p.notifier_id
    AND n.notifier_type <> 'TEAMS';
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM notifiers_mattermost p
USING notifiers n
WHERE n.id = p.notifier_id
    AND n.notifier_type <> 'MATTERMOST';
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM notifiers_webhook p
USING notifiers n
WHERE n.id = p.notifier_id
    AND n.notifier_type <> 'WEBHOOK';
-- +goose StatementEnd

-- A notifier of an unknown type or without its provider row cannot load or send, so it goes
-- together with its attachments to databases.
-- +goose StatementBegin
CREATE TEMPORARY TABLE broken_notifiers ON COMMIT DROP AS
SELECT n.id
FROM notifiers n
WHERE NOT (
        (n.notifier_type = 'EMAIL' AND EXISTS (SELECT 1 FROM notifiers_email p WHERE p.notifier_id = n.id))
        OR (n.notifier_type = 'TELEGRAM' AND EXISTS (SELECT 1 FROM notifiers_telegram p WHERE p.notifier_id = n.id))
        OR (n.notifier_type = 'SLACK' AND EXISTS (SELECT 1 FROM notifiers_slack p WHERE p.notifier_id = n.id))
        OR (n.notifier_type = 'DISCORD' AND EXISTS (SELECT 1 FROM notifiers_discord p WHERE p.notifier_id = n.id))
        OR (n.notifier_type = 'TEAMS' AND EXISTS (SELECT 1 FROM notifiers_teams p WHERE p.notifier_id = n.id))
        OR (n.notifier_type = 'MATTERMOST' AND EXISTS (SELECT 1 FROM notifiers_mattermost p WHERE p.notifier_id = n.id))
        OR (n.notifier_type = 'WEBHOOK' AND EXISTS (SELECT 1 FROM notifiers_webhook p WHERE p.notifier_id = n.id))
    );
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM database_notifiers
WHERE notifier_id IN (SELECT id FROM broken_notifiers);
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM notifiers
WHERE id IN (SELECT id FROM broken_notifiers);
-- +goose StatementEnd

-- A notifier without a workspace joins the workspace of the earliest database that uses it.
-- +goose StatementBegin
UPDATE notifiers n
SET workspace_id = (
    SELECT d.workspace_id
    FROM database_notifiers dn
    JOIN databases d ON d.id = dn.database_id
    WHERE dn.notifier_id = n.id
        AND d.workspace_id IS NOT NULL
    ORDER BY d.id
    LIMIT 1
)
WHERE n.workspace_id IS NULL;
-- +goose StatementEnd

-- A notifier that no database of a workspace uses has no owner left and goes.
-- +goose StatementBegin
DELETE FROM database_notifiers
WHERE notifier_id IN (SELECT id FROM notifiers WHERE workspace_id IS NULL);
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM notifiers
WHERE workspace_id IS NULL;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers
    RENAME COLUMN notifier_type TO type;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers
    ALTER COLUMN type TYPE TEXT;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers
    ALTER COLUMN workspace_id SET NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers
    ADD CONSTRAINT uq_notifiers_id_type
    UNIQUE (id, type);
-- +goose StatementEnd

-- The list index starts with the workspace, so it serves every query this one served.
-- +goose StatementBegin
DROP INDEX idx_notifiers_workspace_id;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_notifiers_workspace_id_name_id
    ON notifiers (workspace_id, name, id);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_email
    DROP CONSTRAINT fk_email_notifiers_notifier;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_email
    ADD COLUMN type TEXT NOT NULL DEFAULT 'EMAIL';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_email
    ADD CONSTRAINT fk_notifiers_email_notifier
    FOREIGN KEY (notifier_id, type)
    REFERENCES notifiers (id, type)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_telegram
    DROP CONSTRAINT fk_telegram_notifiers_notifier;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_telegram
    ADD COLUMN type TEXT NOT NULL DEFAULT 'TELEGRAM';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_telegram
    ADD CONSTRAINT fk_notifiers_telegram_notifier
    FOREIGN KEY (notifier_id, type)
    REFERENCES notifiers (id, type)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_slack
    DROP CONSTRAINT fk_slack_notifiers_notifier;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_slack
    ADD COLUMN type TEXT NOT NULL DEFAULT 'SLACK';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_slack
    ADD CONSTRAINT fk_notifiers_slack_notifier
    FOREIGN KEY (notifier_id, type)
    REFERENCES notifiers (id, type)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_discord
    DROP CONSTRAINT fk_discord_notifiers_notifier;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_discord
    ADD COLUMN type TEXT NOT NULL DEFAULT 'DISCORD';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_discord
    ADD CONSTRAINT fk_notifiers_discord_notifier
    FOREIGN KEY (notifier_id, type)
    REFERENCES notifiers (id, type)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_teams
    DROP CONSTRAINT fk_teams_notifiers_notifier;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_teams
    ADD COLUMN type TEXT NOT NULL DEFAULT 'TEAMS';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_teams
    ADD CONSTRAINT fk_notifiers_teams_notifier
    FOREIGN KEY (notifier_id, type)
    REFERENCES notifiers (id, type)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_mattermost
    DROP CONSTRAINT fk_mattermost_notifiers_notifier;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_mattermost
    ADD COLUMN type TEXT NOT NULL DEFAULT 'MATTERMOST';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_mattermost
    ADD CONSTRAINT fk_notifiers_mattermost_notifier
    FOREIGN KEY (notifier_id, type)
    REFERENCES notifiers (id, type)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_webhook
    DROP CONSTRAINT fk_webhook_notifiers_notifier;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_webhook
    ADD COLUMN type TEXT NOT NULL DEFAULT 'WEBHOOK';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_webhook
    ADD CONSTRAINT fk_notifiers_webhook_notifier
    FOREIGN KEY (notifier_id, type)
    REFERENCES notifiers (id, type)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_webhook
    ALTER COLUMN headers DROP DEFAULT;
-- +goose StatementEnd

-- Header values encrypted by an older release stay inside the JSON as they are.
-- +goose StatementBegin
ALTER TABLE notifiers_webhook
    ALTER COLUMN headers TYPE JSONB
    USING CASE
        WHEN headers IS NULL OR btrim(headers) IN ('', 'null') THEN '[]'::jsonb
        ELSE headers::jsonb
    END;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_webhook
    ALTER COLUMN headers SET DEFAULT '[]';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_webhook
    ALTER COLUMN headers SET NOT NULL;
-- +goose StatementEnd

-- A header without a name cannot be sent and blocks saving the notifier. Copying the kept rows
-- whole leaves their encrypted values as they are.
-- +goose StatementBegin
UPDATE notifiers_webhook
SET headers = jsonb_path_query_array(headers, '$[*] ? (@.key != "")');
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_webhook
    ADD COLUMN accepted_categories TEXT[] NOT NULL DEFAULT '{ALL}';
-- +goose StatementEnd

-- An empty or unknown list accepts every category, as an empty list did before.
-- +goose StatementBegin
UPDATE notifiers_webhook
SET accepted_categories = COALESCE(
    (
        SELECT array_agg(DISTINCT category.value ORDER BY category.value)
        FROM jsonb_array_elements_text(
            CASE
                WHEN btrim(accept_notification_types) IN ('', 'null') THEN '[]'::jsonb
                ELSE accept_notification_types::jsonb
            END
        ) AS category (value)
        WHERE category.value IN ('ALL', 'BACKUP_SUCCESS', 'BACKUP_FAILED', 'HEALTHCHECK_SUCCESS', 'HEALTHCHECK_FAILED', 'VERIFICATION_SUCCESS', 'VERIFICATION_FAILED')
    ),
    '{ALL}'
);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_webhook
    DROP COLUMN accept_notification_types;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE notifiers_webhook
    ADD COLUMN accept_notification_types TEXT NOT NULL DEFAULT '["ALL"]';
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE notifiers_webhook
SET accept_notification_types = to_jsonb(accepted_categories)::text;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_webhook
    DROP COLUMN accepted_categories;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_webhook
    ALTER COLUMN headers DROP NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_webhook
    ALTER COLUMN headers DROP DEFAULT;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_webhook
    ALTER COLUMN headers TYPE TEXT USING headers::text;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_webhook
    ALTER COLUMN headers SET DEFAULT '[]';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_email
    DROP CONSTRAINT fk_notifiers_email_notifier;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_email
    DROP COLUMN type;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_telegram
    DROP CONSTRAINT fk_notifiers_telegram_notifier;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_telegram
    DROP COLUMN type;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_slack
    DROP CONSTRAINT fk_notifiers_slack_notifier;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_slack
    DROP COLUMN type;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_discord
    DROP CONSTRAINT fk_notifiers_discord_notifier;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_discord
    DROP COLUMN type;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_teams
    DROP CONSTRAINT fk_notifiers_teams_notifier;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_teams
    DROP COLUMN type;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_mattermost
    DROP CONSTRAINT fk_notifiers_mattermost_notifier;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_mattermost
    DROP COLUMN type;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_webhook
    DROP CONSTRAINT fk_notifiers_webhook_notifier;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_webhook
    DROP COLUMN type;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX idx_notifiers_workspace_id_name_id;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_notifiers_workspace_id
    ON notifiers (workspace_id);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers
    DROP CONSTRAINT uq_notifiers_id_type;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers
    ALTER COLUMN workspace_id DROP NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers
    ALTER COLUMN type TYPE VARCHAR(50);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers
    RENAME COLUMN type TO notifier_type;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_email
    ADD CONSTRAINT ck_email_notifiers_security
    CHECK (security IN ('tls', 'starttls', 'none'));
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_email
    ADD CONSTRAINT fk_email_notifiers_notifier
    FOREIGN KEY (notifier_id)
    REFERENCES notifiers (id)
    ON DELETE CASCADE
    DEFERRABLE INITIALLY DEFERRED;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_email
    RENAME CONSTRAINT notifiers_email_pkey TO email_notifiers_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_email
    RENAME TO email_notifiers;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_telegram
    ADD CONSTRAINT fk_telegram_notifiers_notifier
    FOREIGN KEY (notifier_id)
    REFERENCES notifiers (id)
    ON DELETE CASCADE
    DEFERRABLE INITIALLY DEFERRED;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_telegram
    RENAME CONSTRAINT notifiers_telegram_pkey TO telegram_notifiers_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_telegram
    RENAME TO telegram_notifiers;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_slack
    ADD CONSTRAINT fk_slack_notifiers_notifier
    FOREIGN KEY (notifier_id)
    REFERENCES notifiers (id)
    ON DELETE CASCADE
    DEFERRABLE INITIALLY DEFERRED;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_slack
    RENAME CONSTRAINT notifiers_slack_pkey TO slack_notifiers_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_slack
    RENAME TO slack_notifiers;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_discord
    ADD CONSTRAINT fk_discord_notifiers_notifier
    FOREIGN KEY (notifier_id)
    REFERENCES notifiers (id)
    ON DELETE CASCADE
    DEFERRABLE INITIALLY DEFERRED;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_discord
    RENAME CONSTRAINT notifiers_discord_pkey TO discord_notifiers_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_discord
    RENAME TO discord_notifiers;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_teams
    ADD CONSTRAINT fk_teams_notifiers_notifier
    FOREIGN KEY (notifier_id)
    REFERENCES notifiers (id)
    ON DELETE CASCADE
    DEFERRABLE INITIALLY DEFERRED;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_teams
    RENAME CONSTRAINT notifiers_teams_pkey TO teams_notifiers_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_teams
    RENAME TO teams_notifiers;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_mattermost
    ADD CONSTRAINT fk_mattermost_notifiers_notifier
    FOREIGN KEY (notifier_id)
    REFERENCES notifiers (id)
    ON DELETE CASCADE
    DEFERRABLE INITIALLY DEFERRED;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_mattermost
    RENAME CONSTRAINT notifiers_mattermost_pkey TO mattermost_notifiers_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_mattermost
    RENAME TO mattermost_notifiers;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_webhook
    ADD CONSTRAINT fk_webhook_notifiers_notifier
    FOREIGN KEY (notifier_id)
    REFERENCES notifiers (id)
    ON DELETE CASCADE
    DEFERRABLE INITIALLY DEFERRED;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_webhook
    RENAME CONSTRAINT notifiers_webhook_pkey TO webhook_notifiers_pkey;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE notifiers_webhook
    RENAME TO webhook_notifiers;
-- +goose StatementEnd
