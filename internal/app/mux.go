package app

import (
	"fmt"
	"net/http"
)

type Controller interface{ Route() *AppMux }

type AppMux struct {
	handlers       map[string]AppHandler
	appMiddlewares []AppMiddleware
	middlewares    []Middleware
	applied        bool
}

func NewAppMux() *AppMux { return &AppMux{handlers: make(map[string]AppHandler)} }

func (m *AppMux) HandleFunc(pattern string, handler AppHandler) {
	m.checkNotApplied()
	if _, exists := m.handlers[pattern]; exists {
		panic(fmt.Sprintf("이미 존재하는 경로입니다: %s", pattern))
	}
	m.handlers[pattern] = handler
}

func (m *AppMux) Handle(pattern string, handler http.Handler) {
	m.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) error { handler.ServeHTTP(w, r); return nil })
}

func (m *AppMux) RegisterAppMiddlewares(middlewares ...AppMiddleware) *AppMux {
	m.checkNotApplied()
	m.appMiddlewares = append(m.appMiddlewares, middlewares...)
	return m
}

func (m *AppMux) RegisterMiddlewares(middlewares ...Middleware) *AppMux {
	m.checkNotApplied()
	m.middlewares = append(m.middlewares, middlewares...)
	return m
}

func (m *AppMux) Nest(child *AppMux) *AppMux {
	m.checkNotApplied()
	if len(child.middlewares) != 0 {
		panic("자식 mux는 net/http Middleware를 가질 수 없습니다")
	}
	child.applyApp()
	for pattern, handler := range child.handlers {
		if _, exists := m.handlers[pattern]; exists {
			panic(fmt.Sprintf("이미 존재하는 경로입니다: %s", pattern))
		}
		m.handlers[pattern] = handler
	}
	return m
}

func (m *AppMux) AddControllers(controllers ...Controller) *AppMux {
	for _, controller := range controllers {
		m.Nest(controller.Route())
	}
	return m
}

func (m *AppMux) Apply() http.Handler {
	m.applyApp()
	mux := http.NewServeMux()
	for pattern, handler := range m.handlers {
		mux.HandleFunc(pattern, AdaptHandler(handler))
	}
	var handler http.Handler = mux
	for i := len(m.middlewares) - 1; i >= 0; i-- {
		handler = m.middlewares[i](handler)
	}
	return handler
}

func (m *AppMux) applyApp() {
	m.checkNotApplied()
	for pattern, handler := range m.handlers {
		for i := len(m.appMiddlewares) - 1; i >= 0; i-- {
			handler = m.appMiddlewares[i](handler)
		}
		m.handlers[pattern] = handler
	}
	m.applied = true
}

func (m *AppMux) checkNotApplied() {
	if m.applied {
		panic("이미 적용된 mux입니다")
	}
}
