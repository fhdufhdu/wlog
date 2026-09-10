package app

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestAppMiddlewareOrder(t *testing.T) {
	var calls []string
	middleware := func(name string) AppMiddleware {
		return func(next AppHandler) AppHandler {
			return func(w http.ResponseWriter, r *http.Request) error {
				calls = append(calls, name+" before")
				err := next(w, r)
				calls = append(calls, name+" after")
				return err
			}
		}
	}
	mux := NewAppMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, _ *http.Request) error {
		calls = append(calls, "handler")
		w.WriteHeader(http.StatusNoContent)
		return nil
	})
	handler := mux.RegisterAppMiddlewares(middleware("first"), middleware("second")).Apply()
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	want := []string{"first before", "second before", "handler", "second after", "first after"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %#v, want %#v", calls, want)
	}
}
