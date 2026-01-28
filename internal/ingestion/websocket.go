package ingestion

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kalshi-signal-feed/internal/config"
	"github.com/kalshi-signal-feed/internal/state"
)

type WebSocketHandler struct {
	url            string
	reconnectDelay time.Duration
	state          *state.Engine
}

func NewWebSocketHandler(cfg config.KalshiConfig, ingestionCfg config.IngestionConfig, stateEngine *state.Engine) *WebSocketHandler {
	return &WebSocketHandler{
		url:            cfg.WebSocketURL,
		reconnectDelay: time.Duration(ingestionCfg.WebSocketReconnectDelaySecs) * time.Second,
		state:          stateEngine,
	}
}

func (w *WebSocketHandler) Run(ctx context.Context) error {
	delay := w.reconnectDelay
	maxDelay := 60 * time.Second

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := w.connectAndListen(ctx)
		if err != nil {
			fmt.Printf("WebSocket error: %v. Reconnecting in %v...\n", err, delay)
		} else {
			fmt.Println("WebSocket connection closed normally")
		}

		// Wait before reconnecting
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}

		// Exponential backoff, max 60 seconds
		delay = delay * 2
		if delay > maxDelay {
			delay = maxDelay
		}
	}
}

func (w *WebSocketHandler) connectAndListen(ctx context.Context) error {
	fmt.Printf("Connecting to WebSocket: %s\n", w.url)

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial(w.url, nil)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	fmt.Println("WebSocket connected")

	// Reset delay on successful connection
	w.reconnectDelay = time.Duration(5) * time.Second

	// Handle messages
	done := make(chan error, 1)

	go func() {
		defer close(done)
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				done <- err
				return
			}
			if err := w.handleMessage(message); err != nil {
				fmt.Printf("Error handling message: %v\n", err)
			}
		}
	}()

	// Send ping periodically
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-done:
			return err
		case <-ticker.C:
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return err
			}
		}
	}
}

func (w *WebSocketHandler) handleMessage(message []byte) error {
	var msg map[string]interface{}
	if err := json.Unmarshal(message, &msg); err != nil {
		return fmt.Errorf("failed to parse message: %w", err)
	}

	msgType, ok := msg["type"].(string)
	if !ok {
		return nil
	}

	switch msgType {
	case "orderbook", "orderbook_update":
		return w.handleOrderbookUpdate(msg)
	case "trade", "trade_update":
		return w.handleTradeUpdate(msg)
	default:
		// Unknown message type, ignore
		return nil
	}
}

func (w *WebSocketHandler) handleOrderbookUpdate(msg map[string]interface{}) error {
	ticker, ok := msg["ticker"].(string)
	if !ok {
		return nil
	}

	// Try to parse orderbook data
	orderbookData, ok := msg["orderbook_fp"].(map[string]interface{})
	if !ok {
		return nil
	}

	yesDollars, _ := orderbookData["yes_dollars"].([]interface{})
	noDollars, _ := orderbookData["no_dollars"].([]interface{})

	// Convert to our format
	orderbook := state.NewOrderbook(ticker)
	orderbookResp := &state.KalshiOrderbookResponse{
		OrderbookFp: state.KalshiOrderbookFp{
			YesDollars: convertToOrderbookLevels(yesDollars),
			NoDollars:  convertToOrderbookLevels(noDollars),
		},
	}

	orderbook.UpdateFromKalshi(orderbookResp)
	w.state.UpdateOrderbook(ticker, orderbook)

	return nil
}

func (w *WebSocketHandler) handleTradeUpdate(msg map[string]interface{}) error {
	ticker, ok := msg["ticker"].(string)
	if !ok {
		return nil
	}

	price, _ := msg["price"].(float64)
	quantity, _ := msg["count"].(float64)
	side, _ := msg["side"].(string)

	trade := &state.Trade{
		MarketTicker: ticker,
		Price:        int(price * 100), // Convert to cents
		Quantity:     int(quantity),
		Timestamp:    time.Now(),
	}

	if side == "yes" {
		trade.Side = state.SideYes
	} else {
		trade.Side = state.SideNo
	}

	w.state.AddTrade(trade)
	return nil
}

func convertToOrderbookLevels(data []interface{}) [][]string {
	result := make([][]string, 0, len(data))
	for _, item := range data {
		if arr, ok := item.([]interface{}); ok {
			level := make([]string, 0, len(arr))
			for _, v := range arr {
				if s, ok := v.(string); ok {
					level = append(level, s)
				} else if f, ok := v.(float64); ok {
					level = append(level, fmt.Sprintf("%.4f", f))
				}
			}
			if len(level) >= 2 {
				result = append(result, level)
			}
		}
	}
	return result
}

