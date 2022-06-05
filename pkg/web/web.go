package web

import (
	"context"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/julienschmidt/httprouter"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
)

type Handler func(ctx context.Context, w http.ResponseWriter, r *http.Request) error

type App struct {
	mux      *httprouter.Router
	otmux    http.Handler
	shutdown chan os.Signal
	mw       []Middleware
}

// NewApp returns a pointer to the App with a configured mux.
func NewApp(shutdown chan os.Signal, mw ...Middleware) *App {
	router := httprouter.New()

	return &App{
		mux:      router,
		otmux:    otelhttp.NewHandler(router, "request"),
		shutdown: shutdown,
		mw:       mw,
	}
}

// ServeHTTP implements the http.Handler interface on App.
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.otmux.ServeHTTP(w, r)
}

func (a *App) Start() error {
	return http.ListenAndServe(":8000", a)
}

// SignalShutdown is used to gracefully shut down the app when an integrity
// issue is identified.
func (a *App) SignalShutdown() {
	a.shutdown <- syscall.SIGTERM
}

// Handle sets a handler function for a given HTTP method and path pair
// to the application server mux.
func (a *App) Handle(method string, group string, path string, handler Handler, mw ...Middleware) {
	// First wrap handler specific middleware around this handler.
	handler = wrapMiddleware(mw, handler)

	// Add the application's general middleware to the handler chain.
	handler = wrapMiddleware(a.mw, handler)

	// The function to execute for each request.
	h := func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		// Pull the context from the request and
		// use it as a separate parameter.
		ctx := r.Context()

		// Capture the parent request span from the context.
		span := trace.SpanFromContext(ctx)

		// Set the context with the required values to
		// process the request.
		v := Values{
			TraceID: span.SpanContext().TraceID().String(),
			Now:     time.Now().UTC(),
		}
		ctx = context.WithValue(ctx, key, &v)

		// Call the wrapped handler functions.
		if err := handler(ctx, w, r); err != nil {
			a.SignalShutdown()
			return
		}
	}

	finalPath := path
	if group != "" {
		finalPath = "/" + group + path
	}

	a.mux.Handle(method, finalPath, h)
}
