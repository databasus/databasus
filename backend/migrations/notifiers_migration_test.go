package migrations_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const notifiersMigrationName = "20261025180000_move_notifiers_into_module.sql"

func TestNotifiersMigration(t *testing.T) {
	t.Parallel()

	t.Run("accepts text values after the inherited email security check is removed", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseMigratedBefore(t, notifiersMigrationName)
		workspaceID := insertWorkspace(t, db)
		insertNotifierOfEveryTypeWithSecrets(t, db, workspaceID)

		applyAllMigrations(t, db, notifiersMigrationName)

		execute(t, db, "UPDATE notifiers_email SET security = 'PROVIDER_DEFINED'")
		assert.Equal(t, []string{"PROVIDER_DEFINED"}, listStrings(t, db, "SELECT security FROM notifiers_email"))
	})

	t.Run("deletes the notifiers that cannot load and the provider rows of another type", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseMigratedBefore(t, notifiersMigrationName)
		workspaceID := insertWorkspace(t, db)
		databaseID := insertDatabase(t, db, &workspaceID)
		unknownTypeNotifierID := insertNotifier(t, db, "PAGER", &workspaceID)
		notifierWithoutRowID := insertNotifier(t, db, "SLACK", &workspaceID)
		attachNotifier(t, db, databaseID, notifierWithoutRowID)
		telegramNotifierID := insertNotifier(t, db, "TELEGRAM", &workspaceID)
		insertTelegramRow(t, db, telegramNotifierID, "enc:telegram-token")
		execute(t, db, "INSERT INTO discord_notifiers (notifier_id, channel_webhook_url) VALUES (?, 'enc:stray')",
			telegramNotifierID)

		applyAllMigrations(t, db, notifiersMigrationName)

		assert.ElementsMatch(t, []uuid.UUID{telegramNotifierID}, listIDs(t, db, "SELECT id FROM notifiers"))
		assert.Empty(t, listIDs(t, db, "SELECT notifier_id FROM notifiers_discord"))
		assert.Empty(t, listIDs(t, db, "SELECT notifier_id FROM databases_notifiers"))
		assert.ElementsMatch(t, []uuid.UUID{databaseID}, listIDs(t, db, "SELECT id FROM databases"))
		assert.NotContains(t, listIDs(t, db, "SELECT id FROM notifiers"), unknownTypeNotifierID)
	})

	t.Run("moves a notifier without a workspace into its database's workspace and deletes an unused one",
		func(t *testing.T) {
			t.Parallel()
			db := openDatabaseMigratedBefore(t, notifiersMigrationName)
			workspaceID := insertWorkspace(t, db)
			databaseID := insertDatabase(t, db, &workspaceID)
			usedNotifierID := insertNotifier(t, db, "TELEGRAM", nil)
			insertTelegramRow(t, db, usedNotifierID, "enc:used")
			attachNotifier(t, db, databaseID, usedNotifierID)
			unusedNotifierID := insertNotifier(t, db, "TELEGRAM", nil)
			insertTelegramRow(t, db, unusedNotifierID, "enc:unused")

			applyAllMigrations(t, db, notifiersMigrationName)

			var movedWorkspaceIDs []uuid.UUID
			require.NoError(t, db.WithContext(t.Context()).Raw(
				"SELECT workspace_id FROM notifiers WHERE id = ?", usedNotifierID,
			).Scan(&movedWorkspaceIDs).Error)
			assert.Equal(t, []uuid.UUID{workspaceID}, movedWorkspaceIDs)
			assert.ElementsMatch(t, []uuid.UUID{usedNotifierID}, listIDs(t, db, "SELECT id FROM notifiers"))
			assert.ElementsMatch(t, []uuid.UUID{usedNotifierID},
				listIDs(t, db, "SELECT notifier_id FROM databases_notifiers"))
		})

	t.Run("converts the webhook headers and accepted categories", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseMigratedBefore(t, notifiersMigrationName)
		workspaceID := insertWorkspace(t, db)
		narrowedNotifierID := insertWebhookNotifier(t, db, workspaceID, webhookColumns{
			headers:         `[{"key":"Authorization","value":"enc:header-secret"}]`,
			acceptedTypes:   `["BACKUP_FAILED","UNKNOWN"]`,
			webhookMethod:   "POST",
			webhookURLValue: "https://hooks.example.com/narrowed",
		})
		emptyNotifierID := insertWebhookNotifier(t, db, workspaceID, webhookColumns{
			headers:         "",
			acceptedTypes:   "[]",
			webhookMethod:   "PATCH",
			webhookURLValue: "https://hooks.example.com/empty",
		})

		applyAllMigrations(t, db, notifiersMigrationName)

		narrowedRow := getWebhookRow(t, db, narrowedNotifierID)
		assert.JSONEq(t, `[{"key":"Authorization","value":"enc:header-secret"}]`, narrowedRow.Headers)
		assert.Equal(t, []string{"BACKUP_FAILED"}, []string(narrowedRow.AcceptedCategories))
		assert.Equal(t, "POST", narrowedRow.WebhookMethod)

		emptyRow := getWebhookRow(t, db, emptyNotifierID)
		assert.JSONEq(t, `[]`, emptyRow.Headers)
		assert.Equal(t, []string{"ALL"}, []string(emptyRow.AcceptedCategories))
		assert.Equal(t, "POST", emptyRow.WebhookMethod)
	})

	t.Run("removes webhook header rows with an empty name and keeps the named ones in order", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseMigratedBefore(t, notifiersMigrationName)
		workspaceID := insertWorkspace(t, db)
		mixedNotifierID := insertWebhookNotifier(t, db, workspaceID, webhookColumns{
			headers: `[{"key":"Authorization","value":"enc:first"},{"key":"","value":"enc:nameless"},` +
				`{"key":"X Team","value":"enc:second"}]`,
			acceptedTypes:   `["ALL"]`,
			webhookMethod:   "POST",
			webhookURLValue: "https://hooks.example.com/mixed",
		})
		namelessNotifierID := insertWebhookNotifier(t, db, workspaceID, webhookColumns{
			headers:         `[{"key":"","value":""}]`,
			acceptedTypes:   `["ALL"]`,
			webhookMethod:   "POST",
			webhookURLValue: "https://hooks.example.com/nameless",
		})

		applyAllMigrations(t, db, notifiersMigrationName)

		assert.JSONEq(t,
			`[{"key":"Authorization","value":"enc:first"},{"key":"X Team","value":"enc:second"}]`,
			getWebhookRow(t, db, mixedNotifierID).Headers)
		assert.JSONEq(t, `[]`, getWebhookRow(t, db, namelessNotifierID).Headers)
	})

	t.Run("keeps every encrypted secret byte for byte", func(t *testing.T) {
		t.Parallel()
		db := openDatabaseMigratedBefore(t, notifiersMigrationName)
		workspaceID := insertWorkspace(t, db)
		secretColumns := insertNotifierOfEveryTypeWithSecrets(t, db, workspaceID)

		applyAllMigrations(t, db, notifiersMigrationName)

		for _, secretColumn := range secretColumns {
			var storedValues []string
			require.NoError(t, db.WithContext(t.Context()).Raw(
				"SELECT "+secretColumn.column+" FROM "+secretColumn.table+" WHERE notifier_id = ?",
				secretColumn.notifierID,
			).Scan(&storedValues).Error)
			assert.Equal(
				t,
				[]string{secretColumn.value},
				storedValues,
				"%s.%s",
				secretColumn.table,
				secretColumn.column,
			)
		}
	})
}

type webhookColumns struct {
	headers         string
	acceptedTypes   string
	webhookMethod   string
	webhookURLValue string
}

type migratedWebhookRow struct {
	Headers            string
	AcceptedCategories pq.StringArray `gorm:"type:text[]"`
	WebhookMethod      string
}

type storedSecret struct {
	notifierID uuid.UUID
	table      string
	column     string
	value      string
}

func insertNotifierOfEveryTypeWithSecrets(t *testing.T, db *gorm.DB, workspaceID uuid.UUID) []storedSecret {
	t.Helper()

	emailNotifierID := insertNotifier(t, db, "EMAIL", &workspaceID)
	execute(t, db, `INSERT INTO email_notifiers
		(notifier_id, target_email, smtp_host, smtp_port, smtp_user, smtp_password, security)
		VALUES (?, 'ops@example.com', 'smtp.example.com', 587, 'ops', 'enc:smtp-password', 'starttls')`,
		emailNotifierID)

	telegramNotifierID := insertNotifier(t, db, "TELEGRAM", &workspaceID)
	execute(t, db, `INSERT INTO telegram_notifiers
		(notifier_id, bot_token, target_chat_id, is_proxy_enabled, proxy_url)
		VALUES (?, 'enc:telegram-token', '42', true, 'enc:telegram-proxy')`, telegramNotifierID)

	slackNotifierID := insertNotifier(t, db, "SLACK", &workspaceID)
	execute(t, db, "INSERT INTO slack_notifiers (notifier_id, bot_token, target_chat_id) VALUES (?, 'enc:slack', 'C1')",
		slackNotifierID)

	discordNotifierID := insertNotifier(t, db, "DISCORD", &workspaceID)
	execute(t, db, "INSERT INTO discord_notifiers (notifier_id, channel_webhook_url) VALUES (?, 'enc:discord')",
		discordNotifierID)

	teamsNotifierID := insertNotifier(t, db, "TEAMS", &workspaceID)
	execute(t, db, "INSERT INTO teams_notifiers (notifier_id, power_automate_url) VALUES (?, 'enc:teams')",
		teamsNotifierID)

	mattermostNotifierID := insertNotifier(t, db, "MATTERMOST", &workspaceID)
	execute(t, db, `INSERT INTO mattermost_notifiers (notifier_id, delivery_mode, webhook_url, bot_token)
		VALUES (?, 'WEBHOOK', 'enc:mattermost-hook', 'enc:mattermost-token')`, mattermostNotifierID)

	webhookNotifierID := insertWebhookNotifier(t, db, workspaceID, webhookColumns{
		headers:         `[{"key":"Authorization","value":"enc:header"}]`,
		acceptedTypes:   `["ALL"]`,
		webhookMethod:   "POST",
		webhookURLValue: "enc:webhook-url",
	})
	execute(t, db, "UPDATE webhook_notifiers SET body_template = 'enc:body' WHERE notifier_id = ?", webhookNotifierID)

	return []storedSecret{
		{notifierID: emailNotifierID, table: "notifiers_email", column: "smtp_password", value: "enc:smtp-password"},
		{notifierID: telegramNotifierID, table: "notifiers_telegram", column: "bot_token", value: "enc:telegram-token"},
		{notifierID: telegramNotifierID, table: "notifiers_telegram", column: "proxy_url", value: "enc:telegram-proxy"},
		{notifierID: slackNotifierID, table: "notifiers_slack", column: "bot_token", value: "enc:slack"},
		{
			notifierID: discordNotifierID,
			table:      "notifiers_discord",
			column:     "channel_webhook_url",
			value:      "enc:discord",
		},
		{notifierID: teamsNotifierID, table: "notifiers_teams", column: "power_automate_url", value: "enc:teams"},
		{
			notifierID: mattermostNotifierID, table: "notifiers_mattermost", column: "webhook_url",
			value: "enc:mattermost-hook",
		},
		{
			notifierID: mattermostNotifierID, table: "notifiers_mattermost", column: "bot_token",
			value: "enc:mattermost-token",
		},
		{notifierID: webhookNotifierID, table: "notifiers_webhook", column: "webhook_url", value: "enc:webhook-url"},
		{notifierID: webhookNotifierID, table: "notifiers_webhook", column: "body_template", value: "enc:body"},
		{
			notifierID: webhookNotifierID, table: "notifiers_webhook", column: "headers->0->>'value'",
			value: "enc:header",
		},
	}
}

func insertNotifier(t *testing.T, db *gorm.DB, notifierType string, workspaceID *uuid.UUID) uuid.UUID {
	t.Helper()

	notifierID := uuid.New()
	execute(t, db, "INSERT INTO notifiers (id, name, notifier_type, workspace_id) VALUES (?, ?, ?, ?)",
		notifierID, "notifier "+notifierID.String(), notifierType, workspaceID)

	return notifierID
}

func insertTelegramRow(t *testing.T, db *gorm.DB, notifierID uuid.UUID, botToken string) {
	t.Helper()

	execute(t, db, "INSERT INTO telegram_notifiers (notifier_id, bot_token, target_chat_id) VALUES (?, ?, '42')",
		notifierID, botToken)
}

func insertWebhookNotifier(t *testing.T, db *gorm.DB, workspaceID uuid.UUID, columns webhookColumns) uuid.UUID {
	t.Helper()

	notifierID := insertNotifier(t, db, "WEBHOOK", &workspaceID)
	execute(t, db, `INSERT INTO webhook_notifiers
		(notifier_id, webhook_url, webhook_method, headers, accept_notification_types)
		VALUES (?, ?, ?, ?, ?)`,
		notifierID, columns.webhookURLValue, columns.webhookMethod, columns.headers, columns.acceptedTypes)

	return notifierID
}

func attachNotifier(t *testing.T, db *gorm.DB, databaseID, notifierID uuid.UUID) {
	t.Helper()

	execute(t, db, "INSERT INTO database_notifiers (database_id, notifier_id) VALUES (?, ?)", databaseID, notifierID)
}

func getWebhookRow(t *testing.T, db *gorm.DB, notifierID uuid.UUID) migratedWebhookRow {
	t.Helper()

	var rows []migratedWebhookRow
	require.NoError(t, db.WithContext(t.Context()).Raw(
		"SELECT headers::text AS headers, accepted_categories, webhook_method FROM notifiers_webhook WHERE notifier_id = ?",
		notifierID,
	).
		Scan(&rows).Error)
	require.Len(t, rows, 1)

	return rows[0]
}
