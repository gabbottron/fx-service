package main

import (
	"context"
	"io"
	"net"
	"net/http"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

// Function to build the HTTP server
func NewHTTPServer(lc fx.Lifecycle, mux *http.ServeMux, log *zap.Logger) *http.Server {
	srv := &http.Server{Addr: ":8080", Handler: mux}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ln, err := net.Listen("tcp", srv.Addr)
			if err != nil {
				return err
			}
			log.Info("Starting HTTP server", zap.String("addr", srv.Addr))
			go srv.Serve(ln)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
	return srv
}

// NewServeMux builds a ServeMux that will route requests
// to the given EchoHandler.
func NewServeMux(echo *EchoHandler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/echo", echo)
	return mux
}

func main() {
	fx.New(
		// Use zap logger for FXs logs as well
		fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: log}
		}),
		// This provides the HTTP Server to the container
		// so that we may use it
		fx.Provide(
			NewHTTPServer,
			NewServeMux,
			NewEchoHandler,
			zap.NewExample, // zap.NewProduction <- for real applications
		),
		// Used for root level invocations like background
		// workers or global loggers. Without this invoke,
		// the lifecycle methods for the HTTP server we set
		// up will not fire with the container. Ensures
		// HTTP server is always instantiated, even if it
		// is not directly referenced in code yet.
		fx.Invoke(func(*http.Server) {}),
	).Run()
}

// EchoHandler is an http.Handler that copies its request body
// back to the response.
type EchoHandler struct {
	log *zap.Logger
}

// NewEchoHandler builds a new EchoHandler.
func NewEchoHandler(log *zap.Logger) *EchoHandler {
	return &EchoHandler{log: log}
}

// ServeHTTP handles an HTTP request to the /echo endpoint.
func (h *EchoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if _, err := io.Copy(w, r.Body); err != nil {
		h.log.Warn("Failed to handle request", zap.Error(err))
	}
}
