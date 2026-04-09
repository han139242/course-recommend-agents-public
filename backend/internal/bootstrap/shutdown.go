package bootstrap

import (
	"context"
	"fmt"
)

// registerShutdownJobs 注册应用退出时需要按顺序执行的清理任务。
//
// 这里采用顺序关闭而不是并发关闭，目的是让 HTTP 服务先停止接收新请求，
// 再逐步释放外部连接，减少停机阶段出现悬空请求或资源竞争的概率。
func (a *App) registerShutdownJobs() {
	a.shutdownJobs = []func(context.Context) error{
		func(ctx context.Context) error {
			return a.HTTPServer.Shutdown(ctx)
		},
	}

	if a.GraphWriter != nil && a.GraphWriter.Service != nil {
		a.shutdownJobs = append(a.shutdownJobs, func(ctx context.Context) error {
			return a.GraphWriter.Service.Stop(ctx)
		})
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
//
// 关闭过程中会尽量执行完全部清理任务，但只返回首个错误，
// 以兼顾资源释放完整性与调用方的错误感知。
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
