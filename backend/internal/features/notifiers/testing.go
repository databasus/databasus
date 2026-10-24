//go:build testhelpers

package notifiers

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"

	webhook_notifier "databasus-backend/internal/features/notifiers/models/webhook"
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

func CreateTestNotifier(workspaceID uuid.UUID) *Notifier {
	notifier := &Notifier{
		WorkspaceID:  workspaceID,
		Name:         "test " + uuid.New().String(),
		NotifierType: NotifierTypeWebhook,
		WebhookNotifier: &webhook_notifier.WebhookNotifier{
			WebhookURL:    GetWebhookStub().URL() + "/test-" + uuid.New().String(),
			WebhookMethod: webhook_notifier.WebhookMethodPOST,
		},
	}

	notifier, err := notifierRepository.Save(notifier)
	if err != nil {
		panic(err)
	}

	return notifier
}

func RemoveTestNotifier(notifier *Notifier) {
	err := notifierRepository.Delete(notifier)
	if err != nil {
		panic(err)
	}
}
