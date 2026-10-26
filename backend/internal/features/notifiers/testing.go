//go:build testhelpers

package notifiers_bridge

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/notifiers"
	"databasus-backend/internal/storage"
)

type WebhookStub struct {
	server    *httptest.Server
	callCount atomic.Int64
}

func startWebhookStub() *WebhookStub {
	stub := &WebhookStub{}

	stub.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		stub.callCount.Add(1)
		w.WriteHeader(http.StatusOK)
	}))

	return stub
}

var sharedWebhookStub = sync.OnceValue(startWebhookStub)

func GetWebhookStub() *WebhookStub {
	return sharedWebhookStub()
}

func (s *WebhookStub) URL() string {
	return s.server.URL
}

func (s *WebhookStub) CallCount() int {
	return int(s.callCount.Load())
}

func (s *WebhookStub) ResetCalls() {
	s.callCount.Store(0)
}

// CreateTestNotifier writes a webhook notifier that posts to the shared webhook stub straight into
// the tables of the notifiers module, because the old tests set up no workspace member to create
// one through the API.
func CreateTestNotifier(workspaceID uuid.UUID) *notifiers.Notifier {
	notifier := &notifiers.Notifier{
		ID:            uuid.New(),
		WorkspaceID:   workspaceID,
		Name:          "test " + uuid.New().String(),
		Type:          notifiers.TypeWebhook,
		LastSendError: nil,
	}

	db := storage.GetDb()

	err := db.Exec(
		"INSERT INTO notifiers (id, workspace_id, name, type) VALUES (?, ?, ?, ?)",
		notifier.ID, notifier.WorkspaceID, notifier.Name, notifier.Type,
	).Error
	if err != nil {
		panic(err)
	}

	err = db.Exec(
		"INSERT INTO notifiers_webhook (notifier_id, webhook_url, webhook_method) VALUES (?, ?, 'POST')",
		notifier.ID, GetWebhookStub().URL()+"/test-"+uuid.New().String(),
	).Error
	if err != nil {
		panic(err)
	}

	return notifier
}

func RemoveTestNotifier(notifier *notifiers.Notifier) {
	err := storage.GetDb().Exec("DELETE FROM notifiers WHERE id = ?", notifier.ID).Error
	if err != nil {
		panic(err)
	}
}
