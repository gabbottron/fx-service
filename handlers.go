package service

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"

	"go.uber.org/zap"
)

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
