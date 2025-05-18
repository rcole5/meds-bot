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

	"meds-bot/internal/config"
	"meds-bot/internal/db"
	"meds-bot/internal/discord"
	"meds-bot/internal/reminder"
)

// run is the main application function that returns the reminder service and any error
func run(ctx context.Context) (reminder.ServiceInterface, error) {
	log.Println("Starting medication reminder bot...")

	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	store, err := db.NewStore(ctx, cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}
	defer func() {
		if ctx.Err() != nil {
			if err := store.Close(); err != nil {
				log.Printf("Error closing database: %v", err)
			}
		}
	}()

	discordClient, err := discord.NewClient(ctx, cfg, store)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Discord client: %w", err)
	}
	defer func() {
		if ctx.Err() != nil {
			if err := discordClient.Close(); err != nil {
				log.Printf("Error closing Discord client: %v", err)
			}
		}
	}()

	reminderService := reminder.NewService(cfg, store, discordClient)

	if err := reminderService.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start reminder service: %w", err)
	}

	// Start health check server
	healthServer := startHealthServer()
	defer func() {
		if ctx.Err() != nil {
			if err := healthServer.Shutdown(ctx); err != nil {
				log.Printf("Error shutting down health server: %v", err)
			}
		}
	}()

	return reminderService, nil
}

// startHealthServer starts a simple HTTP server with health check endpoints
func startHealthServer() *http.Server {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Readiness endpoint
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Ready"))
	})

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("Health server error: %v", err)
		}
	}()

	log.Println("Health check server started on :8080")
	return server
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a channel to track the reminder service
	var reminderService reminder.ServiceInterface
	serviceReady := make(chan reminder.ServiceInterface)

	// Run the application in a goroutine
	go func() {
		service, err := run(ctx)
		if err != nil {
			log.Printf("Application error: %v", err)
			cancel() // Cancel the context to signal shutdown
			serviceReady <- nil
			return
		}
		serviceReady <- service
	}()

	// Wait for the service to be ready
	reminderService = <-serviceReady
	if reminderService == nil {
		log.Println("Failed to start application")
		return
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	fmt.Println("Medication reminder bot is now running. Press CTRL-C to exit.")

	sig := <-sigCh
	log.Printf("Received signal %v, initiating graceful shutdown...", sig)

	// Cancel the context to signal all components to shut down
	cancel()

	// Stop the reminder service explicitly
	if reminderService != nil {
		log.Println("Stopping reminder service...")
		reminderService.Stop()
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Wait for graceful shutdown or timeout
	select {
	case <-time.After(100 * time.Millisecond): // Give a small delay for cleanup
		log.Println("Graceful shutdown completed")
	case <-shutdownCtx.Done():
		if errors.Is(shutdownCtx.Err(), context.DeadlineExceeded) {
			log.Println("Graceful shutdown timed out, forcing exit")
		}
	}
}
