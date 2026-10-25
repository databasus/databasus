package logging

import (
	"log/slog"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_RedactAttr_WithSecretKeyNames_ReplacesValue(t *testing.T) {
	secretKeys := []string{
		"password", "db_passwd", "client_secret", "access_token",
		"api_key", "apikey", "Authorization", "cookie", "aws_credential", "private_key",
	}

	for _, key := range secretKeys {
		redactedAttr := redactAttr(slog.String(key, "hunter2"))

		assert.Equal(t, redactedValue, redactedAttr.Value.String(), "key %q", key)
	}
}

func Test_RedactAttr_WithUrlKey_StripsCredentialsButKeepsHost(t *testing.T) {
	redactedAttr := redactAttr(slog.String("endpoint_url", "https://viewer:hunter2@logs.example.com/v1/logs"))

	assert.Equal(t, "https://redacted@logs.example.com/v1/logs", redactedAttr.Value.String())
}

func Test_RedactAttr_WithKeyValueDsn_RedactsOnlyThePassword(t *testing.T) {
	redactedAttr := redactAttr(slog.String("dsn", "host=localhost user=postgres password=Q1234567 dbname=databasus"))

	assert.Equal(
		t,
		"host=localhost user=postgres password="+redactedValue+" dbname=databasus",
		redactedAttr.Value.String(),
	)
}

func Test_RedactAttr_WithEmailKey_MasksLocalPartAndKeepsDomain(t *testing.T) {
	redactedAttr := redactAttr(slog.String("user_email", "rostislav@acme.com"))

	assert.Equal(t, "r"+redactedValue+"@acme.com", redactedAttr.Value.String())
}

func Test_RedactAttr_WithNestedGroup_RedactsGroupMembers(t *testing.T) {
	redactedAttr := redactAttr(slog.Group("storage",
		slog.String("bucket", "backups"),
		slog.Group("auth", slog.String("secret_key", "sk-live-1234")),
	))

	require.Equal(t, slog.KindGroup, redactedAttr.Value.Kind())

	outerMembers := redactedAttr.Value.Group()
	assert.Equal(t, "backups", outerMembers[0].Value.String())

	innerMembers := outerMembers[1].Value.Group()
	assert.Equal(t, redactedValue, innerMembers[0].Value.String())
}

// "error" is the codebase's most common attr key and url.Error embeds the whole URL.
func Test_RedactAttr_WithErrorCarryingDsn_ScrubsCredentialsFromErrorText(t *testing.T) {
	_, parseErr := url.Parse(
		"postgres://postgres:Q1234567@localhost:5437/databasus\n", //nolint:staticcheck // the invalid URL is the case under test
	)
	require.Error(t, parseErr)

	redactedAttr := redactAttr(slog.Any("error", parseErr))

	assert.NotContains(t, redactedAttr.Value.String(), "Q1234567")
	assert.Contains(t, redactedAttr.Value.String(), "localhost:5437")
}

func Test_RedactAttr_WithNonStringValue_KeepsOriginalKind(t *testing.T) {
	assert.Equal(t, slog.KindInt64, redactAttr(slog.Int("connection_count", 5)).Value.Kind())
	assert.EqualValues(t, 5, redactAttr(slog.Int("connection_count", 5)).Value.Int64())
	assert.Equal(t, slog.KindDuration, redactAttr(slog.Duration("endpoint_timeout", time.Second)).Value.Kind())
}

func Test_RedactAttr_WithArbitraryStringHoldingSecret_ScrubsIt(t *testing.T) {
	redactedAttr := redactAttr(slog.String("command", "mongodump --uri=mongodb://root:hunter2@db:27017"))

	assert.NotContains(t, redactedAttr.Value.String(), "hunter2")
}

// A url-keyed attr whose value is prose rather than a bare URL must not get weaker treatment than
// any other key: url.Parse fails on it, so the fallback has to be the full message scrub.
func Test_RedactAttr_WithUrlKeyHoldingProse_ScrubsEmbeddedCredentials(t *testing.T) {
	redactedAttr := redactAttr(slog.String("endpoint", "connecting to postgres://admin:hunter2@db:5432/app"))

	assert.NotContains(t, redactedAttr.Value.String(), "hunter2")
}

func Test_RedactAttr_WithHarmlessKey_LeavesValueAlone(t *testing.T) {
	redactedAttr := redactAttr(slog.String("database_id", "db-42"))

	assert.Equal(t, "db-42", redactedAttr.Value.String())
}

func Test_RedactMessage_WithUrlCredentials_ReplacesUserAndPassword(t *testing.T) {
	redactedMessage := redactMessage("restoring from mongodb://admin:hunter2@cluster0.example.net/db")

	assert.Equal(
		t,
		"restoring from mongodb://"+redactedValue+":"+redactedValue+"@cluster0.example.net/db",
		redactedMessage,
	)
}

func Test_RedactMessage_WithKeyValueSecret_ReplacesOnlyTheSecret(t *testing.T) {
	redactedMessage := redactMessage("connecting with host=localhost password=Q1234567")

	assert.Equal(t, "connecting with host=localhost password="+redactedValue, redactedMessage)
}

// Audit messages embed raw addresses via fmt.Sprintf, and the dual-write ships them off-box.
func Test_RedactMessage_WithAuditEmailMessage_MasksAddress(t *testing.T) {
	redactedMessage := redactMessage("User deactivated: rostislav@acme.com")

	assert.Equal(t, "User deactivated: r"+redactedValue+"@acme.com", redactedMessage)
}

func Test_RedactMessage_WithNothingSensitive_LeavesMessageAlone(t *testing.T) {
	assert.Equal(t, "backup finished: 12 GB", redactMessage("backup finished: 12 GB"))
}

func Test_RedactMessage_WithBearerHeader_RemovesCredential(t *testing.T) {
	scrubbed := redactMessage("upstream rejected us, sent Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.abc.def")

	assert.NotContains(t, scrubbed, "eyJhbGciOiJIUzI1NiJ9.abc.def")
	assert.Contains(t, scrubbed, "Bearer")
}

func Test_RedactMessage_WithSchemelessTokenHeader_RemovesCredential(t *testing.T) {
	assert.NotContains(t, redactMessage("x-api-key: sk-live-abc123"), "sk-live-abc123")
}

func Test_RedactMessage_WithNumericTokenValue_LeavesItIntact(t *testing.T) {
	counts := []string{
		"replication token: 5 slots",
		"replication token: 5, and more",
		"replication token: 5.",
		"replication token: 5",
	}

	for _, message := range counts {
		assert.Equal(t, message, redactMessage(message))
	}
}

func Test_RedactMessage_WithSecretFollowedByPunctuation_KeepsThePunctuation(t *testing.T) {
	scrubbed := redactMessage("rejected, sent token: sk-live-abc123, retrying")

	assert.NotContains(t, scrubbed, "sk-live-abc123")
	assert.Contains(t, scrubbed, ", retrying")
}

func Test_RedactAttr_WithWebhookUrlKey_ReplacesWholeValue(t *testing.T) {
	redactedAttr := redactAttr(slog.String("webhook_url", "https://discord.com/api/webhooks/999/SECRETTOKEN"))

	assert.Equal(t, redactedValue, redactedAttr.Value.String())
}

func Test_RedactAttr_WithStructValue_RemovesCredential(t *testing.T) {
	credentials := struct {
		Host     string
		Password string
	}{Host: "db.example.com", Password: "hunter2"}

	redactedAttr := redactAttr(slog.Any("connection", credentials))

	assert.NotContains(t, redactedAttr.Value.String(), "hunter2")
	assert.Contains(t, redactedAttr.Value.String(), "db.example.com")
}

func Test_RedactAttr_WithMapValue_RemovesCredential(t *testing.T) {
	headers := map[string]string{"authorization": "Bearer secrettokenvalue"}

	assert.NotContains(t, redactAttr(slog.Any("headers", headers)).Value.String(), "secrettokenvalue")
}

func Test_RedactMessage_WithTimestampsAndRatios_LeavesThemIntact(t *testing.T) {
	untouched := []string{
		"backup completed in 01:23:45",
		"chain cleanup progress: 12 wal, 3 incr, complete=true",
		"listening on 0.0.0.0:4005",
	}

	for _, message := range untouched {
		assert.Equal(t, message, redactMessage(message))
	}
}
