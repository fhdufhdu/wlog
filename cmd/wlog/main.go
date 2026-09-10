package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fhdufhdu/wlog/internal/about"
	"github.com/fhdufhdu/wlog/internal/app"
	"github.com/fhdufhdu/wlog/internal/apperr"
	"github.com/fhdufhdu/wlog/internal/auth"
	"github.com/fhdufhdu/wlog/internal/config"
	"github.com/fhdufhdu/wlog/internal/database"
	images "github.com/fhdufhdu/wlog/internal/image"
	"github.com/fhdufhdu/wlog/internal/post"
	"github.com/fhdufhdu/wlog/internal/topic"
	"github.com/fhdufhdu/wlog/internal/web"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	app.ConfigureLogger(&slog.HandlerOptions{Level: slog.LevelInfo})
	if err := run(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		slog.Warn(".env load failed", "error", err)
	}
	cfg, err := config.FromEnv()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(cfg.UploadDir, 0750); err != nil {
		return err
	}
	if err := database.Migrate(cfg.DatabaseURL, cfg.MigrationBaselineExisting); err != nil {
		return err
	}
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := pool.Ping(context.Background()); err != nil {
		return err
	}
	authService, err := auth.New(cfg.AdminUsername, cfg.AdminPasswordHash, cfg.SessionSecret, cfg.SecureCookie)
	if err != nil {
		return err
	}
	postService := post.NewService(post.NewRepository(pool))
	topicService := topic.NewService(topic.NewRepository(pool))
	aboutService := about.NewService(about.NewRepository(pool))
	imageService := images.NewService(images.NewRepository(pool), cfg.UploadDir, cfg.ImageOrphanGraceHours)
	controller, err := web.NewController(cfg, pool, authService, postService, topicService, aboutService, imageService)
	if err != nil {
		return err
	}
	handler := app.NewAppMux().AddControllers(controller).Nest(controller.StaticRoutes()).RegisterAppMiddlewares(apperr.Middleware()).RegisterMiddlewares(app.RecoverMiddleware(), app.RequestIDMiddleware(), app.SecurityHeadersMiddleware()).Apply()
	server := &http.Server{Addr: cfg.BindAddr, Handler: handler, ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 120 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go cleanupLoop(ctx, imageService)
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}()
	slog.Info("wlog started", "address", cfg.BindAddr)
	err = server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func cleanupLoop(ctx context.Context, service *images.Service) {
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			removed, err := service.Cleanup(ctx)
			if err != nil {
				slog.Error("unused image cleanup failed", "error", err)
			} else if removed > 0 {
				slog.Info("unused images cleaned up", "removed", removed)
			}
		}
	}
}
