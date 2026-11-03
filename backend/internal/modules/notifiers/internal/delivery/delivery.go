// Package delivery sends a notification to notifiers in the background, each with its own timeout,
// and records the outcome of every send.
package delivery

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/notifiers/internal/notification"
	"databasus-backend/internal/modules/notifiers/internal/notifier"
	"databasus-backend/internal/modules/notifiers/internal/providers"
)

// maxMessageRunes is the longest message a notification carries; longer ones are cut.
const maxMessageRunes = 2000

type Deps struct {
	Notifiers *notifier.Service
	Registry  *providers.Registry
	Logger    *slog.Logger
	// Timeout ends a delivery the channel does not answer, which then counts as failed.
	Timeout time.Duration
}

type Delivery struct {
	notifiers *notifier.Service
	registry  *providers.Registry
	logger    *slog.Logger
	timeout   time.Duration
	group     *background.Group
}

func New(deps Deps) *Delivery {
	return &Delivery{
		notifiers: deps.Notifiers,
		registry:  deps.Registry,
		logger:    deps.Logger,
		timeout:   deps.Timeout,
		group:     background.New(background.Deps{Logger: deps.Logger}),
	}
}

// Notify returns at once. The deliveries run on a context that neither the caller's end nor its
// transaction reaches, so the caller notifies only after its transaction commits.
func (d *Delivery) Notify(ctx context.Context, notifierIDs []uuid.UUID, message notification.Notification) {
	uniqueNotifierIDs := slices.Compact(slices.SortedFunc(slices.Values(notifierIDs), compareIDs))
	if len(uniqueNotifierIDs) == 0 {
		return
	}

	messageRunes := []rune(message.Message)
	if len(messageRunes) > maxMessageRunes {
		message.Message = string(messageRunes[:maxMessageRunes])
	}

	deliveryCtx := database.WithoutTransaction(context.WithoutCancel(ctx))

	d.group.Go(deliveryCtx, "notification deliveries", func(ctx context.Context) error {
		recipients, err := d.notifiers.ListForDelivery(ctx, uniqueNotifierIDs)
		if err != nil {
			return fmt.Errorf("load the notifiers to deliver to: %w", err)
		}

		for _, recipient := range recipients {
			d.group.Go(ctx, "notification delivery", func(ctx context.Context) error {
				return d.deliver(ctx, recipient, message)
			})
		}

		return nil
	})
}

// Wait returns once every started delivery has ended, or with ctx's error.
func (d *Delivery) Wait(ctx context.Context) error {
	return d.group.Wait(ctx)
}

func (d *Delivery) deliver(
	ctx context.Context,
	recipient notifier.WithSettings,
	message notification.Notification,
) error {
	sendErr := d.send(ctx, recipient, message)
	if sendErr != nil {
		d.logger.ErrorContext(ctx, "failed to send notification",
			slog.String("notifier_id", recipient.ID.String()),
			slog.String("notification_category", string(message.Category)),
			slog.Any("error", sendErr),
		)
	}

	err := d.notifiers.RecordSendOutcome(ctx, recipient.ID, sendErr)
	if err != nil {
		return fmt.Errorf("record the outcome of the delivery to notifier %s: %w", recipient.ID, err)
	}

	return nil
}

func (d *Delivery) send(
	ctx context.Context,
	recipient notifier.WithSettings,
	message notification.Notification,
) error {
	provider, err := d.registry.GetProvider(recipient.Type)
	if err != nil {
		return err
	}

	timeout := d.timeout
	if longerProvider, isLonger := provider.(providers.DeliveryTimeoutProvider); isLonger {
		timeout = longerProvider.GetDeliveryTimeout()
	}

	sendCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	sendErr := provider.Send(sendCtx, recipient.Settings, message)
	if sendErr != nil && errors.Is(sendCtx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("the delivery timed out after %s: %w", timeout, sendErr)
	}

	return sendErr
}

func compareIDs(first, second uuid.UUID) int {
	return slices.Compare(first[:], second[:])
}
