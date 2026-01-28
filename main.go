package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/kalshi-signal-feed/internal/alerting"
	"github.com/kalshi-signal-feed/internal/api"
	"github.com/kalshi-signal-feed/internal/config"
	"github.com/kalshi-signal-feed/internal/ingestion"
	"github.com/kalshi-signal-feed/internal/signals"
	"github.com/kalshi-signal-feed/internal/state"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting Kalshi Signal Feed System")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	log.Println("Configuration loaded")

	// Initialize state engine
	stateEngine := state.NewEngine()
	log.Println("State engine initialized")

	// Create signal channel
	signalChan := make(chan signals.Signal, 100)

	// Initialize signal processor
	signalProcessor := signals.NewProcessor(stateEngine, signalChan, cfg.Signals)
	log.Println("Signal processor initialized")

	// Initialize alert manager
	alertManager := alerting.NewManager(cfg.Alerting, signalChan)
	log.Println("Alert manager initialized")

	// Initialize ingestion layer
	ingestionLayer, err := ingestion.NewLayer(cfg.Kalshi, cfg.Ingestion, stateEngine)
	if err != nil {
		log.Fatalf("Failed to initialize ingestion layer: %v", err)
	}
	log.Println("Ingestion layer initialized")

	// Initialize API server
	apiServer := api.NewServer(cfg.API, stateEngine, signalChan)
	log.Println("API server initialized")

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start all components
	var wg sync.WaitGroup

	// Start ingestion
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := ingestionLayer.Run(ctx); err != nil {
			log.Printf("Ingestion layer error: %v", err)
		}
	}()

	// Start signal processor
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := signalProcessor.Run(ctx); err != nil {
			log.Printf("Signal processor error: %v", err)
		}
	}()

	// Start alert manager
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := alertManager.Run(ctx); err != nil {
			log.Printf("Alert manager error: %v", err)
		}
	}()

	// Start API server
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := apiServer.Run(ctx); err != nil {
			log.Printf("API server error: %v", err)
		}
	}()

	log.Println("All components started. System running...")

	// Wait for interrupt signal
	<-sigChan
	log.Println("Shutting down...")

	// Cancel context to stop all components
	cancel()

	// Wait for all components to finish
	wg.Wait()
	log.Println("Shutdown complete")
}

