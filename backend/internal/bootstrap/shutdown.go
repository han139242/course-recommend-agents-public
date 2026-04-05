package bootstrap

import (
	"context"
	"fmt"
)

func (a *App) registerShutdownJobs() {
	a.shutdownJobs = []func(context.Context) error{
		func(ctx context.Context) error {
			return a.HTTPServer.Shutdown(ctx)
		},
	}

	if a.Redis != nil {
		a.shutdownJobs = append(a.shutdownJobs, func(_ context.Context) error {
			return a.Redis.Close()
		})
	}

	if a.Neo4j != nil {
		a.shutdownJobs = append(a.shutdownJobs, func(ctx context.Context) error {
			return a.Neo4j.Close(ctx)
		})
	}

	if a.MySQL != nil {
		a.shutdownJobs = append(a.shutdownJobs, func(_ context.Context) error {
			sqlDB, err := a.MySQL.DB()
			if err != nil {
				return err
			}
			return sqlDB.Close()
		})
	}
}

// Shutdown 按注册顺序关闭应用依赖。
func (a *App) Shutdown(ctx context.Context) error {
	var firstErr error

	for _, job := range a.shutdownJobs {
		if err := job(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	if firstErr != nil {
		return fmt.Errorf("应用关闭失败: %w", firstErr)
	}

	return nil
}
