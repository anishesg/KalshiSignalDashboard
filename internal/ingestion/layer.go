package ingestion

import (
	"context"
	"fmt"
	"time"

	"github.com/kalshi-signal-feed/internal/config"
	"github.com/kalshi-signal-feed/internal/state"
)

type Layer struct {
	restClient  *RESTClient
	wsHandler   *WebSocketHandler
	state       *state.Engine
	pollInterval time.Duration
}

func NewLayer(kalshiCfg config.KalshiConfig, ingestionCfg config.IngestionConfig, stateEngine *state.Engine) (*Layer, error) {
	restClient, err := NewRESTClient(kalshiCfg, ingestionCfg, stateEngine)
	if err != nil {
		return nil, fmt.Errorf("failed to create REST client: %w", err)
	}

	wsHandler := NewWebSocketHandler(kalshiCfg, ingestionCfg, stateEngine)

	return &Layer{
		restClient:   restClient,
		wsHandler:    wsHandler,
		state:        stateEngine,
		pollInterval: time.Duration(ingestionCfg.RESTPollIntervalSecs) * time.Second,
	}, nil
}

func (l *Layer) Run(ctx context.Context) error {
	// Start WebSocket handler
	go func() {
		if err := l.wsHandler.Run(ctx); err != nil && err != context.Canceled {
			fmt.Printf("WebSocket handler error: %v\n", err)
		}
	}()

	// Start orderbook polling
	go func() {
		l.PollOrderbooks(ctx)
	}()

	// Start REST polling for markets
	if err := l.restClient.PollMarkets(ctx); err != nil && err != context.Canceled {
		return fmt.Errorf("REST client error: %w", err)
	}

	return nil
}

// PollOrderbooks periodically fetches orderbooks for active markets
func (l *Layer) PollOrderbooks(ctx context.Context) {
	// Fetch immediately on startup, then periodically
	l.fetchAllOrderbooks(ctx)
	
	ticker := time.NewTicker(l.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			l.fetchAllOrderbooks(ctx)
		}
	}
}

func (l *Layer) fetchAllOrderbooks(ctx context.Context) {
	markets := l.state.GetAllMarkets()
	activeCount := 0
	successCount := 0
	
	for _, market := range markets {
		if market.Status != state.StatusActive {
			continue
		}
		activeCount++

		// Use a context with timeout for each orderbook fetch
		fetchCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		orderbook, err := l.restClient.GetOrderbook(fetchCtx, market.Ticker)
		cancel()
		
		if err != nil {
			// Only log errors occasionally to avoid spam
			if activeCount%10 == 0 {
				fmt.Printf("Error fetching orderbook for %s: %v\n", market.Ticker, err)
			}
			continue
		}

		ob := state.NewOrderbook(market.Ticker)
		ob.UpdateFromKalshi(orderbook)
		l.state.UpdateOrderbook(market.Ticker, ob)
		successCount++
	}
	
	if activeCount > 0 {
		fmt.Printf("Orderbook poll: %d/%d active markets updated\n", successCount, activeCount)
	}
}

