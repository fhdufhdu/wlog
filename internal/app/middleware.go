package app

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"runtime/debug"
	"time"

	"github.com/google/uuid"
)

type contextKey string

const requestIDKey contextKey = "request-id"

func RequestID(ctx context.Context) string {
	value, _ := ctx.Value(requestIDKey).(string)
	return value
}

func ConfigureLogger(options *slog.HandlerOptions) *slog.Logger {
	logger := slog.New(&LogHandler{Handler: slog.NewTextHandler(os.Stdout, options)})
	slog.SetDefault(logger)
	return logger
}

type LogHandler struct{ slog.Handler }

func (h *LogHandler) Handle(ctx context.Context, record slog.Record) error {
	if id := RequestID(ctx); id != "" {
		record.AddAttrs(slog.String("request_id", id))
	}
	return h.Handler.Handle(ctx, record)
}
func (h *LogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &LogHandler{Handler: h.Handler.WithAttrs(attrs)}
}
func (h *LogHandler) WithGroup(name string) slog.Handler {
	return &LogHandler{Handler: h.Handler.WithGroup(name)}
}

func RequestIDMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := uuid.NewString()
			w.Header().Set("X-Request-Id", id)
			r = r.WithContext(context.WithValue(r.Context(), requestIDKey, id))
			start := time.Now()
			next.ServeHTTP(w, r)
			slog.InfoContext(r.Context(), "request", "request_id", id, "method", r.Method, "path", r.URL.Path, "remote", r.RemoteAddr, "duration", time.Since(start))
		})
	}
}

func RecoverMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if value := recover(); value != nil {
					slog.ErrorContext(r.Context(), "panic", "error", value, "stack", string(debug.Stack()))
					http.Error(w, "서버에서 요청을 처리하지 못했습니다.", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func SecurityHeadersMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			next.ServeHTTP(w, r)
		})
	}
}
