package app

import "net/http"

type AppHandler func(http.ResponseWriter, *http.Request) error
type AppMiddleware func(AppHandler) AppHandler
type Middleware func(http.Handler) http.Handler

func AdaptHandler(handler AppHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) { _ = handler(w, r) }
}
