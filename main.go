package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net"
	"net/http"

	_ "github.com/lib/pq" // PostgreSQL driver

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

// Route is an http.Handler that knows the mux pattern
// under which it will be registered.
type Route interface {
	http.Handler

	// Pattern reports the path at which this is registered.
	Pattern() string
}

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
// to the given Routes.
func NewServeMux(routes []Route) *http.ServeMux {
	mux := http.NewServeMux()
	for _, route := range routes {
		mux.Handle(route.Pattern(), route)
	}
	return mux
}

// Function to initialize the PostgreSQL connection
func NewPostgresConnection(lc fx.Lifecycle, log *zap.Logger) (*sql.DB, error) {
	connStr := "user=username dbname=mydb sslmode=disable" // Replace with your actual connection string
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	// Register lifecycle hooks to close the connection properly
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			log.Info("Connecting to PostgreSQL")
			return db.Ping()
		},
		OnStop: func(context.Context) error {
			log.Info("Closing PostgreSQL connection")
			return db.Close()
		},
	})

	return db, nil
}

// AsRoute annotates the given constructor to state that
// it provides a route to the "routes" group.
func AsRoute(f any) any {
	return fx.Annotate(
		f,
		fx.As(new(Route)),
		fx.ResultTags(`group:"routes"`),
	)
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
			// annotate NewServeMux to say that it accepts a slice that contains the contents of the "routes" group
			fx.Annotate(
				NewServeMux,
				fx.ParamTags(`group:"routes"`),
			),
			// handlers should be provided as a Route
			// this feeds their routes into this group
			AsRoute(NewEchoHandler),
			AsRoute(NewHelloHandler),
			NewPostgresConnection, // Provide the PostgreSQL connection
			zap.NewExample,        // zap.NewProduction <- for real applications
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
	db  *sql.DB
}

// NewEchoHandler builds a new EchoHandler.
func NewEchoHandler(log *zap.Logger, db *sql.DB) *EchoHandler {
	return &EchoHandler{log: log, db: db}
}

// EchoHandler implements the Route interface
func (*EchoHandler) Pattern() string {
	return "/echo"
}

// ServeHTTP handles an HTTP request to the /echo endpoint.
func (h *EchoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if _, err := io.Copy(w, r.Body); err != nil {
		h.log.Warn("Failed to handle request", zap.Error(err))
	}
}

// HelloHandler is an HTTP handler that
// prints a greeting to the user.
type HelloHandler struct {
	log *zap.Logger
	db  *sql.DB
}

// NewHelloHandler builds a new HelloHandler.
func NewHelloHandler(log *zap.Logger, db *sql.DB) *HelloHandler {
	return &HelloHandler{log: log, db: db}
}

func (*HelloHandler) Pattern() string {
	return "/hello"
}

func (h *HelloHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.log.Error("Failed to read request", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if _, err := fmt.Fprintf(w, "Hello, %s\n", body); err != nil {
		h.log.Error("Failed to write response", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}
