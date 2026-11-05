// Package module implements the notifiers.Module contract on top of the module's services.
package module

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/modules/notifiers/internal/delivery"
	notifiershttpapi "databasus-backend/internal/modules/notifiers/internal/httpapi"
	"databasus-backend/internal/modules/notifiers/internal/notification"
	"databasus-backend/internal/modules/notifiers/internal/notifier"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

type Deps struct {
	Notifiers *notifier.Service
	Transfers *notifier.TransferService
	Delivery  *delivery.Delivery
	Handler   *notifiershttpapi.Handler
}

type Module struct {
	notifiers *notifier.Service
	transfers *notifier.TransferService
	delivery  *delivery.Delivery
	handler   *notifiershttpapi.Handler
}

func New(deps Deps) *Module {
	return &Module{
		notifiers: deps.Notifiers,
		transfers: deps.Transfers,
		delivery:  deps.Delivery,
		handler:   deps.Handler,
	}
}

func (m *Module) RegisterRoutes(routes gin.IRouter, hooks *apierror.Hooks) {
	notifiershttpapi.RegisterHandlersWithOptions(
		routes,
		notifiershttpapi.NewStrictHandlerWithOptions(m.handler, nil, notifiershttpapi.StrictGinServerOptions{
			RequestErrorHandlerFunc:  hooks.HandleRequestError,
			HandlerErrorFunc:         hooks.HandleHandlerError,
			ResponseErrorHandlerFunc: hooks.HandleResponseError,
		}),
		notifiershttpapi.GinServerOptions{BaseURL: "", Middlewares: nil, ErrorHandler: hooks.HandleParameterError},
	)
}

func (m *Module) Notify(ctx context.Context, notifierIDs []uuid.UUID, message notification.Notification) {
	m.delivery.Notify(ctx, notifierIDs, message)
}

func (m *Module) GetNotifiers(ctx context.Context, notifierIDs []uuid.UUID) ([]notifier.Notifier, error) {
	return m.notifiers.GetNotifiers(ctx, notifierIDs)
}

func (m *Module) GetNotifierTypes(ctx context.Context) ([]notification.NotifierType, error) {
	return m.notifiers.GetNotifierTypes(ctx)
}

func (m *Module) TransferNotifier(ctx context.Context, actor users.Actor, transfer notifier.Transfer) error {
	return m.transfers.TransferNotifier(ctx, actor, transfer)
}

func (m *Module) OnWorkspaceDeleting(ctx context.Context, event workspaces.WorkspaceDeleting) error {
	return m.notifiers.OnWorkspaceDeleting(ctx, event)
}

func (m *Module) SubscribeToNotifierTransferring(handler notifier.TransferringHandler) {
	m.transfers.SubscribeToTransferring(handler)
}

func (m *Module) WaitForDeliveries(ctx context.Context) error {
	return m.delivery.Wait(ctx)
}
