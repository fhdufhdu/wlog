package apperr

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/fhdufhdu/wlog/internal/app"
	"github.com/jackc/pgx/v5/pgconn"
)

type Kind int

const (
	Internal Kind = iota
	Validation
	Unauthorized
	NotFound
	Conflict
)

type Error struct {
	Kind    Kind
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}
func (e *Error) Unwrap() error { return e.Cause }

func New(kind Kind, message string) error { return &Error{Kind: kind, Message: message} }
func Wrap(cause error, message string) error {
	return &Error{Kind: Internal, Message: message, Cause: cause}
}

func PGCode(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}

func Middleware() app.AppMiddleware {
	return func(next app.AppHandler) app.AppHandler {
		return func(w http.ResponseWriter, r *http.Request) error {
			if err := next(w, r); err != nil {
				status, message := http.StatusInternalServerError, "서버에서 요청을 처리하지 못했습니다."
				var appErr *Error
				if errors.As(err, &appErr) {
					switch appErr.Kind {
					case Validation:
						status, message = http.StatusBadRequest, appErr.Message
					case Unauthorized:
						status, message = http.StatusUnauthorized, "로그인이 필요합니다."
					case NotFound:
						status, message = http.StatusNotFound, "요청한 페이지를 찾을 수 없습니다."
					case Conflict:
						status, message = http.StatusConflict, appErr.Message
					}
				}
				if status >= 500 {
					slog.ErrorContext(r.Context(), "request failed", "error", err, "stack", string(debug.Stack()))
				}
				http.Error(w, message, status)
			}
			return nil
		}
	}
}
