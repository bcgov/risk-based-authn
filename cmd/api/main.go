package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"rba/internal/server"
	"rba/rules"

	"github.com/joho/godotenv"
)

func gracefulShutdown(apiServer *http.Server, done chan bool) {
	// Create context that listens for the interrupt signal from the OS.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Listen for the interrupt signal.
	<-ctx.Done()

	log.Println("shutting down gracefully, press Ctrl+C again to force")
	stop() // Allow Ctrl+C to force shutdown

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := apiServer.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown with error: %v", err)
	}

	log.Println("Server exiting")

	// Notify the main goroutine that the shutdown is complete
	done <- true
}

/*
Loads authentication secrets from environment variables into a map
*/
func loadSecrets() (map[string][]byte, error) {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, skipping...")
	}

	// Build a map of API keys -> secrets
	secrets := map[string][]byte{}

	for _, id := range []string{"CLIENT_1", "CLIENT_2"} {
		key := os.Getenv(fmt.Sprintf("API_KEY_%s", id))
		secret := os.Getenv(fmt.Sprintf("API_SECRET_%s", id))
		if key != "" && secret != "" {
			secrets[key] = []byte(secret)
		} else {
			return nil, errors.New("could not load expected api keys")
		}
	}

	return secrets, nil
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found")
	}

	handlers, serviceConfig, rulesConfig, err := rules.LoadConfig("./rules.yaml")

	if err != nil {
		panic(err)
	}

	authKeys, err := loadSecrets()
	if err != nil {
		log.Fatalf("failed to load secrets")
	}

	server := server.NewServer(handlers, serviceConfig, rulesConfig, authKeys)

	// Create a done channel to signal when the shutdown is complete
	done := make(chan bool, 1)
	go gracefulShutdown(server, done)

	err = server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		panic(fmt.Sprintf("http server error: %s", err))
	}

	<-done
	log.Println("Graceful shutdown complete.")
}
