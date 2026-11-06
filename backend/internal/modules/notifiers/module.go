// Package notifiers owns the notification channels of workspaces, their provider settings, their
// test sends and the delivery of notifications. Other modules reach it only through this file.
package notifiers

import (
	"context"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/notifiers/internal/delivery"
	notifiershttpapi "databasus-backend/internal/modules/notifiers/internal/httpapi"
	"databasus-backend/internal/modules/notifiers/internal/module"
	"databasus-backend/internal/modules/notifiers/internal/notification"
	"databasus-backend/internal/modules/notifiers/internal/notifier"
	"databasus-backend/internal/modules/notifiers/internal/providers"
	"databasus-backend/internal/modules/notifiers/internal/providers/discord"
	"databasus-backend/internal/modules/notifiers/internal/providers/email"
	"databasus-backend/internal/modules/notifiers/internal/providers/mattermost"
	"databasus-backend/internal/modules/notifiers/internal/providers/slack"
	"databasus-backend/internal/modules/notifiers/internal/providers/teams"
	"databasus-backend/internal/modules/notifiers/internal/providers/telegram"
	"databasus-backend/internal/modules/notifiers/internal/providers/webhook"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

type (
	// Type is the channel a notifier sends through. It is fixed when the notifier is created.
	Type = notification.NotifierType
	// Category says what a notification is about; a webhook notifier can accept only some
	// categories.
	Category = notification.Category
	// Notification is a message about an event of a database.
	Notification = notification.Notification
	// Notifier is a notification channel as other modules see it, without its settings.
	Notifier = notifier.Notifier
	// NotifierTransfer names the notifier to move and the workspace it moves to.
	NotifierTransfer = notifier.Transfer
	// NotifierTransferring is published inside the transfer's transaction, before the notifier
	// moves.
	NotifierTransferring = notifier.Transferring
	// NotifierTransferringHandler runs in the transfer's transaction; an error refuses the transfer.
	NotifierTransferringHandler = notifier.TransferringHandler
	// Endpoints are the addresses of the services a provider calls when its settings name none.
	Endpoints = providers.Endpoints
)

const (
	CategoryBackupSuccess       = notification.CategoryBackupSuccess
	CategoryBackupFailed        = notification.CategoryBackupFailed
	CategoryHealthcheckSuccess  = notification.CategoryHealthcheckSuccess
	CategoryHealthcheckFailed   = notification.CategoryHealthcheckFailed
	CategoryVerificationSuccess = notification.CategoryVerificationSuccess
	CategoryVerificationFailed  = notification.CategoryVerificationFailed
)

type Module interface {
	RegisterRoutes(routes gin.IRouter, hooks *apierror.Hooks)

	// Notify returns at once; each notifier gets its own delivery in the background. Call it after
	// the caller's transaction commits, because a sent message cannot be taken back.
	Notify(ctx context.Context, notifierIDs []uuid.UUID, notification Notification)
	// GetNotifiers leaves out the IDs that match no notifier.
	GetNotifiers(ctx context.Context, notifierIDs []uuid.UUID) ([]Notifier, error)
	// GetNotifierTypes returns the type of every notifier of the instance, for telemetry.
	GetNotifierTypes(ctx context.Context) ([]Type, error)
	TransferNotifier(ctx context.Context, actor users.Actor, transfer NotifierTransfer) error
	OnWorkspaceDeleting(ctx context.Context, event workspaces.WorkspaceDeleting) error
	// SubscribeToNotifierTransferring is called by internal/app while it builds the application.
	// The handlers run in the order they subscribed.
	SubscribeToNotifierTransferring(handler NotifierTransferringHandler)
	// WaitForDeliveries returns once every started delivery has ended, or with ctx's error.
	WaitForDeliveries(ctx context.Context) error
}

type Deps struct {
	DB             *gorm.DB
	Transactor     *database.Transactor
	Logger         *slog.Logger
	AuditLog       *audit.Log
	FieldEncryptor *crypto.FieldEncryptor
	Users          users.Module
	Workspaces     workspaces.Module
	// DeliveryTimeout ends a send the channel does not answer, a delivery or a test. A provider may
	// declare a longer limit for its deliveries.
	DeliveryTimeout time.Duration
	Endpoints       Endpoints
	// PublicURL is the address of the instance, from which the default SMTP greeting takes its
	// host; empty when the operator set none.
	PublicURL string
}

func New(deps Deps) Module {
	registry := providers.NewRegistry(
		email.New(email.Deps{
			Repositories:   email.NewRepositories(deps.DB, deps.FieldEncryptor),
			FieldEncryptor: deps.FieldEncryptor,
			PublicURL:      deps.PublicURL,
		}),
		telegram.New(telegram.Deps{
			Repositories:      telegram.NewRepositories(deps.DB, deps.FieldEncryptor),
			FieldEncryptor:    deps.FieldEncryptor,
			OfficialBotAPIURL: deps.Endpoints.TelegramBotAPIURL,
		}),
		slack.New(slack.Deps{
			Repositories:   slack.NewRepositories(deps.DB, deps.FieldEncryptor),
			FieldEncryptor: deps.FieldEncryptor,
			PostMessageURL: deps.Endpoints.SlackPostMessageURL,
		}),
		discord.New(discord.Deps{
			Repositories:   discord.NewRepositories(deps.DB, deps.FieldEncryptor),
			FieldEncryptor: deps.FieldEncryptor,
		}),
		teams.New(teams.Deps{
			Repositories:   teams.NewRepositories(deps.DB, deps.FieldEncryptor),
			FieldEncryptor: deps.FieldEncryptor,
		}),
		mattermost.New(mattermost.Deps{
			Repositories:   mattermost.NewRepositories(deps.DB, deps.FieldEncryptor),
			FieldEncryptor: deps.FieldEncryptor,
		}),
		webhook.New(webhook.Deps{
			Repositories:   webhook.NewRepositories(deps.DB, deps.FieldEncryptor),
			FieldEncryptor: deps.FieldEncryptor,
		}),
	)

	repositories := notifier.NewRepositories(deps.DB)

	notifierService := notifier.NewService(notifier.Deps{
		Repositories: repositories,
		Registry:     registry,
		Transactor:   deps.Transactor,
		AuditLog:     deps.AuditLog,
		Users:        deps.Users,
		Workspaces:   deps.Workspaces,
	})

	transferService := notifier.NewTransferService(notifier.TransferServiceDeps{
		Notifiers:    notifierService,
		Repositories: repositories,
		Transactor:   deps.Transactor,
		AuditLog:     deps.AuditLog,
		Users:        deps.Users,
		Workspaces:   deps.Workspaces,
	})

	return module.New(module.Deps{
		Notifiers: notifierService,
		Transfers: transferService,
		Delivery: delivery.New(delivery.Deps{
			Notifiers: notifierService,
			Registry:  registry,
			Logger:    deps.Logger,
			Timeout:   deps.DeliveryTimeout,
		}),
		Handler: notifiershttpapi.NewHandler(notifiershttpapi.Deps{
			Notifiers: notifierService,
			Transfers: transferService,
			TestSends: notifier.NewTestSendService(notifier.TestSendServiceDeps{
				Notifiers:  notifierService,
				Registry:   registry,
				Workspaces: deps.Workspaces,
				Timeout:    deps.DeliveryTimeout,
			}),
		}),
	})
}
