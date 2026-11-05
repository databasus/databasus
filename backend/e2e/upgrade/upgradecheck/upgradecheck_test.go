package upgradecheck_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/e2e/upgrade/upgradecheck"
)

const webhookURL = "http://upgrade-webhook.invalid/hook"

func TestCompare(t *testing.T) {
	t.Parallel()

	t.Run("returns no error when every written item is listed with its values", func(t *testing.T) {
		t.Parallel()
		writtenItems := []upgradecheck.Item{newWebhookNotifier(webhookURL)}
		listedItems := []upgradecheck.Item{
			newWebhookNotifier(webhookURL),
			{Kind: "storage", Name: "added by the current code", Values: nil},
		}

		require.NoError(t, upgradecheck.Compare(writtenItems, listedItems))
	})

	t.Run("names the notifier and the value when a migration emptied the webhook URL", func(t *testing.T) {
		t.Parallel()
		writtenItems := []upgradecheck.Item{newWebhookNotifier(webhookURL)}
		listedItems := []upgradecheck.Item{newWebhookNotifier("")}

		err := upgradecheck.Compare(writtenItems, listedItems)

		require.ErrorIs(t, err, upgradecheck.ErrItemsDiffer)
		assert.Contains(t, err.Error(), `notifier "upgrade webhook": webhookUrl is "" after the upgrade`)
		assert.Contains(t, err.Error(), webhookURL)
	})

	t.Run("names the item when it is not listed after the upgrade", func(t *testing.T) {
		t.Parallel()
		writtenItems := []upgradecheck.Item{newWebhookNotifier(webhookURL)}

		err := upgradecheck.Compare(writtenItems, nil)

		require.ErrorIs(t, err, upgradecheck.ErrItemsDiffer)
		assert.Contains(t, err.Error(), `notifier "upgrade webhook" is not listed after the upgrade`)
	})

	t.Run("names the value when the listed item no longer has it", func(t *testing.T) {
		t.Parallel()
		writtenItems := []upgradecheck.Item{newWebhookNotifier(webhookURL)}
		listedItems := []upgradecheck.Item{{Kind: "notifier", Name: "upgrade webhook", Values: map[string]string{}}}

		err := upgradecheck.Compare(writtenItems, listedItems)

		require.ErrorIs(t, err, upgradecheck.ErrItemsDiffer)
		assert.Contains(t, err.Error(), `notifier "upgrade webhook": webhookUrl is missing after the upgrade`)
	})
}

func newWebhookNotifier(webhookURL string) upgradecheck.Item {
	return upgradecheck.Item{
		Kind:   "notifier",
		Name:   "upgrade webhook",
		Values: map[string]string{"webhookUrl": webhookURL},
	}
}
