package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/han139242/course-recommend-agents/backend/internal/bootstrap"
	"github.com/han139242/course-recommend-agents/backend/pkg/ctxmeta"
	applogger "github.com/han139242/course-recommend-agents/backend/pkg/logger"
)

// main 是后端服务入口。
// 启动流程负责完成依赖装配、HTTP 服务启动以及优雅停机控制。
func main() {
	ctx := context.Background()

	app, err := bootstrap.New(ctx)
	if err != nil {
		log.Fatalf("应用初始化失败: %v", err)
	}

	startupCtx := ctxmeta.WithTraceID(context.Background(), "system-startup")
	app.Logger.WithCtx(startupCtx).Info("应用启动中", applogger.String("addr", app.HTTPServer.Addr), applogger.String("env", app.Config.App.Env), applogger.String("name", app.Config.App.Name))

	go func() {
		serveCtx := ctxmeta.WithTraceID(context.Background(), "system-http-server")
		if err := app.HTTPServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			app.Logger.WithCtx(serveCtx).Error("HTTP 服务启动失败", applogger.ErrorField("error", err))
			os.Exit(1)
		}
	}()

	// 主协程只负责等待系统信号，并驱动统一停机流程。
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	shutdownLogCtx := ctxmeta.WithTraceID(context.Background(), "system-shutdown")
	app.Logger.WithCtx(shutdownLogCtx).Info("收到退出信号，开始优雅停机")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), app.Config.HTTP.ShutdownTimeout)
	defer cancel()

	if err := app.Shutdown(shutdownCtx); err != nil {
		app.Logger.WithCtx(shutdownLogCtx).Error("应用关闭失败", applogger.ErrorField("error", err))
		os.Exit(1)
	}

	app.Logger.WithCtx(shutdownLogCtx).Info("应用已安全退出", applogger.String("time", time.Now().Format(time.RFC3339)))
}
