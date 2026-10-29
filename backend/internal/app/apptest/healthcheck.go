package apptest

import "context"

func (a *App) RunJobPassWithContext(ctx context.Context, name string) error {
	return a.app.RunJobPass(ctx, name)
}
