package notifier

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/notifiers/internal/notification"
	"databasus-backend/internal/modules/notifiers/internal/providers"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

type TestSendServiceDeps struct {
	Notifiers  *Service
	Registry   *providers.Registry
	Workspaces workspaces.Module
	// Timeout ends a test send that the channel does not answer.
	Timeout time.Duration
}

// SettingsTest is a test of settings that are not saved. NotifierID names the saved notifier whose
// stored secrets fill the ones the settings leave out, nil for a new notifier.
type SettingsTest struct {
	WorkspaceID uuid.UUID
	NotifierID  *uuid.UUID
	Settings    providers.Settings
}

// TestSendService sends test messages. Neither test holds a transaction while the channel answers.
type TestSendService struct {
	notifiers  *Service
	registry   *providers.Registry
	workspaces workspaces.Module
	timeout    time.Duration
}

func NewTestSendService(deps TestSendServiceDeps) *TestSendService {
	return &TestSendService{
		notifiers:  deps.Notifiers,
		registry:   deps.Registry,
		workspaces: deps.Workspaces,
		timeout:    deps.Timeout,
	}
}

// TestNotifier sends a test message with the stored settings and records the outcome as the
// notifier's last send error, as a scheduled notification does.
func (s *TestSendService) TestNotifier(
	ctx context.Context,
	actor users.Actor,
	notifierID uuid.UUID,
) (TestResult, error) {
	testedNotifier, err := s.notifiers.getAuthorized(ctx, actor, notifierID, workspaces.ActionView)
	if err != nil {
		return TestResult{}, err
	}

	testedNotifierWithSettings, err := s.notifiers.loadOneWithSettings(ctx, testedNotifier)
	if err != nil {
		return TestResult{}, err
	}

	sendErr := s.sendTestMessage(ctx, testedNotifierWithSettings.Settings)

	err = s.notifiers.RecordSendOutcome(ctx, notifierID, sendErr)
	if err != nil {
		return TestResult{}, err
	}

	return newTestResult(sendErr), nil
}

// TestSettings sends a test message with the given settings and saves nothing. It needs the right to
// change the workspace's content, because the test sends to wherever the settings point, and a saved
// notifier's stored secrets fill in while the resource address stays the stored one. A named notifier of another workspace is
// answered as a missing one before anything is sent.
func (s *TestSendService) TestSettings(
	ctx context.Context,
	actor users.Actor,
	settingsTest SettingsTest,
) (TestResult, error) {
	err := s.workspaces.Authorize(ctx, actor, settingsTest.WorkspaceID, workspaces.ActionEditContent)
	if err != nil {
		return TestResult{}, err
	}

	testedSettings, err := s.prepareSettings(ctx, settingsTest)
	if err != nil {
		return TestResult{}, err
	}

	return newTestResult(s.sendTestMessage(ctx, testedSettings)), nil
}

func (s *TestSendService) prepareSettings(ctx context.Context, settingsTest SettingsTest) (providers.Settings, error) {
	if settingsTest.NotifierID == nil {
		provider, err := s.registry.GetProvider(settingsTest.Settings.GetType())
		if err != nil {
			return nil, err
		}

		return settingsTest.Settings, provider.Validate(settingsTest.Settings)
	}

	storedNotifier, isFound, err := s.notifiers.repository.get(ctx, *settingsTest.NotifierID)
	if err != nil {
		return nil, err
	}

	if !isFound || storedNotifier.WorkspaceID != settingsTest.WorkspaceID {
		return nil, ErrNotifierNotFound
	}

	if storedNotifier.Type != settingsTest.Settings.GetType() {
		return nil, ErrTypeChanged
	}

	return s.notifiers.mergeAndValidate(ctx, storedNotifier, settingsTest.Settings)
}

// sendTestMessage returns why the channel did not take the message, nil when it did.
func (s *TestSendService) sendTestMessage(ctx context.Context, settings providers.Settings) error {
	provider, err := s.registry.GetProvider(settings.GetType())
	if err != nil {
		return err
	}

	sendCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	sendErr := provider.Send(sendCtx, settings, notification.Notification{
		Category: notification.CategoryAll,
		Heading:  "Test message",
		Message:  "This is a test message",
	})
	if sendErr != nil && errors.Is(sendCtx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("the test send timed out after %s: %w", s.timeout, sendErr)
	}

	return sendErr
}

func newTestResult(sendErr error) TestResult {
	if sendErr != nil {
		return TestResult{IsDelivered: false, Error: sendErr.Error()}
	}

	return TestResult{IsDelivered: true, Error: ""}
}
