package ingestion

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/kalshi-signal-feed/internal/config"
	"github.com/kalshi-signal-feed/internal/state"
	"golang.org/x/time/rate"
)

type RESTClient struct {
	baseURL     string
	auth        *Auth
	client      *http.Client
	state       *state.Engine
	rateLimiter *rate.Limiter
}

type GetMarketsResponse struct {
	Markets []KalshiMarket `json:"markets"`
	Cursor  *string        `json:"cursor"`
}

type KalshiMarket struct {
	Ticker         string  `json:"ticker"`
	Title          string  `json:"title"`
	Category       string  `json:"category"`
	Status         string  `json:"status"`
	ExpirationTime *string `json:"expiration_time"`
	EventTicker    string  `json:"event_ticker"`
	YesSubTitle    string  `json:"yes_sub_title,omitempty"`
	NoSubTitle     string  `json:"no_sub_title,omitempty"`
}

func NewRESTClient(cfg config.KalshiConfig, ingestionCfg config.IngestionConfig, stateEngine *state.Engine) (*RESTClient, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	var auth *Auth
	if cfg.APIKeyID != "" && cfg.PrivateKeyPath != "" {
		privateKeyPEM, err := os.ReadFile(cfg.PrivateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read private key: %w", err)
		}

		auth, err = NewAuth(cfg.APIKeyID, string(privateKeyPEM))
		if err != nil {
			return nil, fmt.Errorf("failed to initialize auth: %w", err)
		}
	}

	rateLimiter := rate.NewLimiter(rate.Limit(ingestionCfg.RateLimitPerSecond), ingestionCfg.RateLimitPerSecond)

	return &RESTClient{
		baseURL:     cfg.APIBaseURL,
		auth:        auth,
		client:      client,
		state:       stateEngine,
		rateLimiter: rateLimiter,
	}, nil
}

func (c *RESTClient) PollMarkets(ctx context.Context) error {
	// First, fetch all politics series
	fmt.Println("Fetching politics series...")
	politicsSeries, err := c.fetchPoliticsSeries(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch politics series: %w", err)
	}
	fmt.Printf("Found %d politics series\n", len(politicsSeries))

	// Poll markets for each series periodically
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Fetch markets for each politics series
		for _, seriesTicker := range politicsSeries {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			// Wait for rate limit
			if err := c.rateLimiter.Wait(ctx); err != nil {
				return err
			}

			var cursor *string
			for {
				resp, err := c.fetchMarkets(ctx, &seriesTicker, cursor)
				if err != nil {
					fmt.Printf("Error fetching markets for series %s: %v\n", seriesTicker, err)
					break
				}

				// Register markets in state
				for _, m := range resp.Markets {
					market := &state.Market{
						Ticker:      m.Ticker,
						Title:       m.Title,
						Category:    m.Category,
						Status:      parseMarketStatus(m.Status),
						EventTicker: m.EventTicker,
						YesSubTitle: m.YesSubTitle,
						NoSubTitle:  m.NoSubTitle,
					}

					if m.ExpirationTime != nil {
						if t, err := time.Parse(time.RFC3339, *m.ExpirationTime); err == nil {
							market.ExpirationTime = &t
						}
					}

					c.state.RegisterMarket(market)
				}

				cursor = resp.Cursor
				if cursor == nil || *cursor == "" {
					break
				}
			}
		}

		// Wait before next full poll cycle
		fmt.Printf("Completed market poll cycle, waiting 60s...\n")
		time.Sleep(60 * time.Second)
	}
}

func (c *RESTClient) fetchMarkets(ctx context.Context, seriesTicker *string, cursor *string) (*GetMarketsResponse, error) {
	url := c.baseURL + "/markets"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Set("limit", "100")
	q.Set("status", "open") // Only open markets
	if seriesTicker != nil {
		q.Set("series_ticker", *seriesTicker)
	}
	if cursor != nil {
		q.Set("cursor", *cursor)
	}
	req.URL.RawQuery = q.Encode()

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch markets: status %d, body: %s", resp.StatusCode, string(body))
	}

	var marketsResp GetMarketsResponse
	if err := json.NewDecoder(resp.Body).Decode(&marketsResp); err != nil {
		return nil, err
	}

	return &marketsResp, nil
}

func (c *RESTClient) GetOrderbook(ctx context.Context, ticker string) (*state.KalshiOrderbookResponse, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/markets/%s/orderbook", ticker)
	url := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Add authentication if available
	if c.auth != nil {
		headers, err := c.auth.SignRequest("GET", path, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("KALSHI-ACCESS-KEY", headers.AccessKey)
		req.Header.Set("KALSHI-ACCESS-SIGNATURE", headers.AccessSignature)
		req.Header.Set("KALSHI-ACCESS-TIMESTAMP", headers.AccessTimestamp)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch orderbook: status %d, body: %s", resp.StatusCode, string(body))
	}

	var orderbookResp state.KalshiOrderbookResponse
	if err := json.NewDecoder(resp.Body).Decode(&orderbookResp); err != nil {
		return nil, err
	}

	return &orderbookResp, nil
}

func parseMarketStatus(s string) state.MarketStatus {
	switch s {
	case "initialized":
		return state.StatusInitialized
	case "inactive":
		return state.StatusInactive
	case "active":
		return state.StatusActive
	case "closed":
		return state.StatusClosed
	case "determined":
		return state.StatusDetermined
	case "disputed":
		return state.StatusDisputed
	case "amended":
		return state.StatusAmended
	case "finalized":
		return state.StatusFinalized
	default:
		return state.StatusInactive
	}
}


