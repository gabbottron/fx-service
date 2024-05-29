package service

import (
	"context"
	"database/sql"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// DbConfig holds the configuration parameters
type DbConfig struct {
	PostgresConnString string
}

// Function to initialize the PostgreSQL connection
func NewPostgresConnection(lc fx.Lifecycle, log *zap.Logger, config DbConfig) (*sql.DB, error) {
	db, err := sql.Open("postgres", config.PostgresConnString)
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
