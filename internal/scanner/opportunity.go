package scanner

import (
	"sort"
	"time"

	"github.com/kalshi-signal-feed/internal/state"
)

// MarketOpportunity represents a market with actionable metrics
type MarketOpportunity struct {
	MarketTicker string    `json:"market_ticker"`
	Title         string    `json:"title"`
	Status        string    `json:"status"`

	// Top-of-book
	BestBid      int     `json:"best_bid"`      // cents
	BestAsk      int     `json:"best_ask"`      // cents
	MidPrice     float64 `json:"mid_price"`     // probability (0-100)
	Spread       int     `json:"spread"`        // cents
	SpreadPercent float64 `json:"spread_percent"` // percentage points

	// Depth metrics
	BidDepth     int64   `json:"bid_depth"`     // total depth in cents
	AskDepth     int64   `json:"ask_depth"`     // total depth in cents
	DepthAtTop5  int64   `json:"depth_at_top5"` // contracts at top 5 levels
	LiquidityScore float64 `json:"liquidity_score"` // 0-1

	// Activity metrics
	RecentTrades    int       `json:"recent_trades"`     // count in last 30s
	LastTradePrice  *int      `json:"last_trade_price"` // cents
	LastTradeTime   *time.Time `json:"last_trade_time"`
	TradeIntensity  float64   `json:"trade_intensity"`   // trades per minute

	// Volatility
	Volatility30s  float64 `json:"volatility_30s"`  // price change in last 30s
	PriceChange30s  float64 `json:"price_change_30s"` // percentage points

	// Microstructure
	Imbalance      float64 `json:"imbalance"`       // -1 to +1
	Microprice     float64 `json:"microprice"`      // probability (0-100)
	MicropriceDiff float64 `json:"microprice_diff"` // microprice - mid

	// Staleness
	LastUpdate     time.Time `json:"last_update"`
	Staleness      float64   `json:"staleness"`     // seconds since last update
	BookStale      bool      `json:"book_stale"`    // >5s since update

	// Execution metrics
	EstimatedSlippage100 int     `json:"estimated_slippage_100"` // cents for 100 contracts
	CanExecute100        bool    `json:"can_execute_100"`       // sufficient depth
}

// Scanner analyzes markets and identifies opportunities
type Scanner struct {
	state *state.Engine
}

func NewScanner(stateEngine *state.Engine) *Scanner {
	return &Scanner{
		state: stateEngine,
	}
}

// ScanMarkets analyzes all active markets and returns opportunities
func (s *Scanner) ScanMarkets() []MarketOpportunity {
	markets := s.state.GetAllMarkets()
	var opportunities []MarketOpportunity

	for _, market := range markets {
		if market.Status != state.StatusActive {
			continue
		}

		opp := s.analyzeMarket(market.Ticker, market.Title, string(market.Status))
		if opp != nil {
			opportunities = append(opportunities, *opp)
		}
	}

	// Sort by liquidity score (best first)
	sort.Slice(opportunities, func(i, j int) bool {
		return opportunities[i].LiquidityScore > opportunities[j].LiquidityScore
	})

	return opportunities
}

func (s *Scanner) analyzeMarket(ticker, title, status string) *MarketOpportunity {
	orderbook, exists := s.state.GetOrderbook(ticker)
	if !exists || len(orderbook.Bids) == 0 || len(orderbook.Asks) == 0 {
		return nil
	}

	opp := &MarketOpportunity{
		MarketTicker: ticker,
		Title:        title,
		Status:       status,
		LastUpdate:   orderbook.LastUpdate,
		Staleness:    time.Since(orderbook.LastUpdate).Seconds(),
		BookStale:    time.Since(orderbook.LastUpdate) > 5*time.Second,
	}

	// Top-of-book
	opp.BestBid = orderbook.Bids[0].Price
	opp.BestAsk = orderbook.Asks[0].Price
	opp.MidPrice = float64(opp.BestBid+opp.BestAsk) / 200.0
	opp.Spread = opp.BestAsk - opp.BestBid
	opp.SpreadPercent = float64(opp.Spread) / 100.0

	// Depth
	opp.BidDepth = orderbook.BidDepth()
	opp.AskDepth = orderbook.AskDepth()
	bidDepth5, askDepth5 := orderbook.DepthAtPrice(5) // within 5 cents
	opp.DepthAtTop5 = bidDepth5 + askDepth5

	// Liquidity score (0-1): based on tight spread and good depth
	spreadScore := 1.0 - (float64(opp.Spread) / 100.0) // tighter is better
	if spreadScore < 0 {
		spreadScore = 0
	}
	depthScore := float64(opp.DepthAtTop5) / 1000.0 // normalize
	if depthScore > 1.0 {
		depthScore = 1.0
	}
	opp.LiquidityScore = (spreadScore*0.6 + depthScore*0.4)

	// Microstructure
	opp.Imbalance = orderbook.ImbalanceRatio()
	if microprice, ok := orderbook.Microprice(); ok {
		opp.Microprice = microprice * 100.0
		opp.MicropriceDiff = opp.Microprice - opp.MidPrice
	}

	// Recent trades
	recentTrades := s.state.GetRecentTrades(ticker, 30*time.Second)
	opp.RecentTrades = len(recentTrades)
	if len(recentTrades) > 0 {
		lastTrade := recentTrades[len(recentTrades)-1]
		opp.LastTradePrice = &lastTrade.Price
		opp.LastTradeTime = &lastTrade.Timestamp
		opp.TradeIntensity = float64(len(recentTrades)) / 0.5 // trades per minute
	}

	// Volatility (price change in last 30s)
	ts := s.state.GetTimeSeries()
	if priceChange, ok := ts.GetPriceChange(ticker, 30*time.Second); ok {
		opp.PriceChange30s = priceChange
		opp.Volatility30s = priceChange // Simplified
	}

	// Execution metrics
	opp.EstimatedSlippage100 = s.estimateSlippage(orderbook, 100)
	opp.CanExecute100 = opp.DepthAtTop5 >= 100 && opp.Spread < 50 // reasonable spread

	return opp
}

// estimateSlippage estimates slippage for executing Q contracts
func (s *Scanner) estimateSlippage(orderbook *state.Orderbook, quantity int) int {
	// Simulate walking the book
	remaining := quantity
	totalCost := 0
	avgPrice := 0.0

	// Walk bids (if selling)
	for _, level := range orderbook.Bids {
		if remaining <= 0 {
			break
		}
		fillQty := remaining
		if fillQty > level.Quantity {
			fillQty = level.Quantity
		}
		totalCost += level.Price * fillQty
		remaining -= fillQty
	}

	if remaining > 0 {
		// Not enough depth, estimate worst case
		return 10000 // 100% slippage (can't fill)
	}

	avgPrice = float64(totalCost) / float64(quantity)
	midPrice := float64(orderbook.Bids[0].Price+orderbook.Asks[0].Price) / 2.0
	slippage := int(avgPrice - midPrice)

	if slippage < 0 {
		slippage = -slippage
	}

	return slippage
}


