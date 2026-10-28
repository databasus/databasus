package backuping_physical

import (
	"context"

	"github.com/google/uuid"
)

func (b *PhysicalBackuper) Dispatch(ctx context.Context, backupID uuid.UUID, isCallNotifier bool) {
	b.dispatchedBackups.Go(func() {
		b.MakeBackup(ctx, backupID, isCallNotifier)
	})
}

func (b *PhysicalBackuper) WaitForDispatchedBackups(ctx context.Context) error {
	isDone := make(chan struct{})
	go func() {
		b.dispatchedBackups.Wait()
		close(isDone)
	}()

	select {
	case <-isDone:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
