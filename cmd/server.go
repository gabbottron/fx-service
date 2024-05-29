package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	_ "github.com/lib/pq" // PostgreSQL driver

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"

	service "github.com/gabbottron/fx-service"
)

var (
	fPort = flag.String("port", os.Getenv("PORT"), "the port the service will listen on")

	fDBName     = flag.String("dbname", os.Getenv("DB_NAME"), "the name of the database to connect to")
	fDBHost     = flag.String("dbhost", os.Getenv("DB_HOSTNAME"), "the hostname of the database to connect to")
	fDBPort     = flag.String("dbport", os.Getenv("DB_PORT"), "the port of the database to connect to")
	fDBUsername = flag.String("dbusername", os.Getenv("DB_USERNAME"), "the user for the database")
	fDBPassword = flag.String("dbpassword", os.Getenv("DB_PASSWORD"), "the password for the database")
)

// ProvideDbConfig provides the configuration for the persistence layer
func ProvideDbConfig() service.DbConfig {
	return service.DbConfig{
		PostgresConnString: fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%s sslmode=disable", *fDBUsername, *fDBPassword, *fDBName, *fDBHost, *fDBPort),
	}
}

// ProvideTransportConfig provides the configuration for the transport layer
func ProvideTransportConfig() service.TransportConfig {
	return service.TransportConfig{
		Port: *fPort, // Replace with the desired port
	}
}

func main() {
	// parse command line flags and environment variables
	flag.Parse()

	fx.New(
		// Use zap logger for FXs logs as well
		fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: log}
		}),
		// This provides the HTTP Server to the container
		// so that we may use it
		fx.Provide(
			service.NewHTTPServer,
			// annotate NewServeMux to say that it accepts a slice that contains the contents of the "routes" group
			fx.Annotate(
				service.NewServeMux,
				fx.ParamTags(`group:"routes"`),
			),
			// handlers should be provided as a Route
			// this feeds their routes into this group
			service.AsRoute(service.NewEchoHandler),
			service.AsRoute(service.NewHelloHandler),
			service.NewPostgresConnection, // Provide the PostgreSQL connection
			ProvideDbConfig,
			ProvideTransportConfig,
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
