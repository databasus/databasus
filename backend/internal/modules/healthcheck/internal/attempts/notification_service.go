package attempts

import (
	"context"
	"fmt"

	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/notifiers"
)

func (s *Service) notify(ctx context.Context, userDatabase databases.Database, status databases.HealthStatus) {
	notification := notifiers.Notification{
		Category: notifiers.CategoryHealthcheckFailed,
		Heading:  fmt.Sprintf("❌ [%s] DB is unavailable", userDatabase.Name),
		Message:  fmt.Sprintf("❌ [%s] DB is currently unavailable", userDatabase.Name),
	}

	if status == databases.HealthStatusAvailable {
		notification = notifiers.Notification{
			Category: notifiers.CategoryHealthcheckSuccess,
			Heading:  fmt.Sprintf("✅ [%s] DB is online", userDatabase.Name),
			Message:  fmt.Sprintf("✅ [%s] DB is back online", userDatabase.Name),
		}
	}
	s.notifiers.Notify(ctx, userDatabase.NotifierIDs, notification)
}
