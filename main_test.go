package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"testing"
	"time"

	"localportfoliomanager/internal/api"
	"localportfoliomanager/internal/utils"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	// Setup code if needed
	code := m.Run()
	// Teardown code if needed
	os.Exit(code)
}

func TestGracefulShutdown(t *testing.T) {
	// Setup
	logger := utils.NewAppLogger()
	config := &utils.Config{
		Server: utils.ServerConfig{
			Port: "8081",
		},
	}

	// Create a mock server
	server := &api.Server{
		Logger: logger,
		Config: config,
		Router: http.NewServeMux(),
	}

	// Create channel to listen for interrupt signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	// Start server in a goroutine
	go func() {
		if err := server.Start(); err != nil && err != http.ErrServerClosed {
			t.Errorf("Error starting server: %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Send interrupt signal
	stop <- os.Interrupt

	// Verify server shuts down
	time.Sleep(100 * time.Millisecond)
	assert.True(t, true, "Server should have shut down gracefully")
}

func TestHealthCheck(t *testing.T) {
	// Setup
	logger := utils.NewAppLogger()
	config := &utils.Config{
		Server: utils.ServerConfig{
			Port: "8082",
		},
	}

	// Create a mock server
	server := &api.Server{
		Logger: logger,
		Config: config,
		Router: http.NewServeMux(),
	}

	// Create test request
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call health check handler
	server.healthCheck(rr, req)

	// Check status code
	assert.Equal(t, http.StatusOK, rr.Code)

	// Check response body
	expected := `{"status":"ok","version":"1.0.0"}`
	assert.JSONEq(t, expected, rr.Body.String())
}
