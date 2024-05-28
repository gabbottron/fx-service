package main

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"go.uber.org/fx"
)

// Function to build the HTTP server
func NewHTTPServer(lc fx.Lifecycle) *http.Server {
	srv := &http.Server{Addr: ":8080"}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ln, err := net.Listen("tcp", srv.Addr)
			if err != nil {
				return err
			}
			fmt.Println("Starting HTTP server at", srv.Addr)
			go srv.Serve(ln)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
	return srv
}

func main() {
	fx.New(
		// This provides the HTTP Server to the container
		// so that we may use it
		fx.Provide(NewHTTPServer),
		// Used for root level invocations like background
		// workers or global loggers. Without this invoke,
		// the lifecycle methods for the HTTP server we set
		// up will not fire with the container. Ensures
		// HTTP server is always instantiated, even if it
		// is not directly referenced in code yet.
		fx.Invoke(func(*http.Server) {}),
	).Run()
}
