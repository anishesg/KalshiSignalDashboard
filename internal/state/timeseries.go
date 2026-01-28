package state

import (
	"sync"
	"time"
)

// TimeSeriesPoint represents a single data point in a time series
type TimeSeriesPoint struct {
	Timestamp time.Time
	Value     float64
	Metadata  map[string]interface{}
}

// MarketSnapshot represents a complete market state at a point in time
type MarketSnapshot struct {
	Timestamp    time.Time
	MarketTicker string
	BestBid      int     // cents
	BestAsk      int     // cents
	MidPrice     float64 // probability (0-100)
	Spread       int     // cents
	BidDepth     int64
	AskDepth     int64
	Imbalance    float64 // -1 to +1
	Microprice   float64 // probability
	TradeCount   int
	LastTrade    *Trade
}

// TimeSeriesStore maintains historical data for backtesting and analysis
type TimeSeriesStore struct {
	mu sync.RWMutex

	// Market snapshots (top-of-book + depth + imbalance)
	snapshots map[string][]MarketSnapshot // market_ticker -> []snapshot

	// Trade history
	trades map[string][]*Trade // market_ticker -> []trade

	// Signal history
	signals map[string][]SignalPoint // market_ticker -> []signal

	// Configuration
	maxSnapshotsPerMarket int
	maxTradesPerMarket    int
	maxSignalsPerMarket   int
}

type SignalPoint struct {
	Timestamp time.Time
	Type      string
	Value     float64
	Metadata  map[string]interface{}
}

func NewTimeSeriesStore() *TimeSeriesStore {
	return &TimeSeriesStore{
		snapshots:             make(map[string][]MarketSnapshot),
		trades:                make(map[string][]*Trade),
		signals:               make(map[string][]SignalPoint),
		maxSnapshotsPerMarket: 10000, // ~2.7 hours at 1s intervals
		maxTradesPerMarket:    10000,
		maxSignalsPerMarket:   10000,
	}
}

// RecordSnapshot records a market snapshot
func (ts *TimeSeriesStore) RecordSnapshot(ticker string, orderbook *Orderbook, trades []*Trade) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if len(orderbook.Bids) == 0 || len(orderbook.Asks) == 0 {
		return
	}

	bestBid := orderbook.Bids[0].Price
	bestAsk := orderbook.Asks[0].Price
	midPrice := float64(bestBid+bestAsk) / 200.0 // Convert to probability
	spread := bestAsk - bestBid

	microprice, _ := orderbook.Microprice()
	micropriceProb := microprice * 100.0 // Convert to percentage

	snapshot := MarketSnapshot{
		Timestamp:    time.Now(),
		MarketTicker: ticker,
		BestBid:      bestBid,
		BestAsk:      bestAsk,
		MidPrice:     midPrice,
		Spread:       spread,
		BidDepth:     orderbook.BidDepth(),
		AskDepth:     orderbook.AskDepth(),
		Imbalance:    orderbook.ImbalanceRatio(),
		Microprice:   micropriceProb,
		TradeCount:   len(trades),
	}

	if len(trades) > 0 {
		snapshot.LastTrade = trades[len(trades)-1]
	}

	snapshots := ts.snapshots[ticker]
	snapshots = append(snapshots, snapshot)

	// Keep only recent snapshots
	if len(snapshots) > ts.maxSnapshotsPerMarket {
		snapshots = snapshots[len(snapshots)-ts.maxSnapshotsPerMarket:]
	}

	ts.snapshots[ticker] = snapshots
}

// RecordTrade records a trade
func (ts *TimeSeriesStore) RecordTrade(ticker string, trade *Trade) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	trades := ts.trades[ticker]
	trades = append(trades, trade)

	if len(trades) > ts.maxTradesPerMarket {
		trades = trades[len(trades)-ts.maxTradesPerMarket:]
	}

	ts.trades[ticker] = trades
}

// RecordSignal records a signal
func (ts *TimeSeriesStore) RecordSignal(ticker string, signalType string, value float64, metadata map[string]interface{}) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	signals := ts.signals[ticker]
	signals = append(signals, SignalPoint{
		Timestamp: time.Now(),
		Type:      signalType,
		Value:     value,
		Metadata:  metadata,
	})

	if len(signals) > ts.maxSignalsPerMarket {
		signals = signals[len(signals)-ts.maxSignalsPerMarket:]
	}

	ts.signals[ticker] = signals
}

// GetSnapshots returns snapshots for a market within a time window
func (ts *TimeSeriesStore) GetSnapshots(ticker string, since time.Time) []MarketSnapshot {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	snapshots := ts.snapshots[ticker]
	var filtered []MarketSnapshot

	for _, s := range snapshots {
		if s.Timestamp.After(since) || s.Timestamp.Equal(since) {
			filtered = append(filtered, s)
		}
	}

	return filtered
}

// GetRecentSnapshots returns the N most recent snapshots
func (ts *TimeSeriesStore) GetRecentSnapshots(ticker string, n int) []MarketSnapshot {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	snapshots := ts.snapshots[ticker]
	if len(snapshots) <= n {
		return snapshots
	}

	return snapshots[len(snapshots)-n:]
}

// GetTrades returns trades for a market within a time window
func (ts *TimeSeriesStore) GetTrades(ticker string, since time.Time) []*Trade {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	trades := ts.trades[ticker]
	var filtered []*Trade

	for _, t := range trades {
		if t.Timestamp.After(since) || t.Timestamp.Equal(since) {
			filtered = append(filtered, t)
		}
	}

	return filtered
}

// GetVolatility computes price volatility over a time window
func (ts *TimeSeriesStore) GetVolatility(ticker string, window time.Duration) float64 {
	since := time.Now().Add(-window)
	snapshots := ts.GetSnapshots(ticker, since)

	if len(snapshots) < 2 {
		return 0
	}

	var prices []float64
	for _, s := range snapshots {
		prices = append(prices, s.MidPrice)
	}

	// Compute standard deviation
	mean := 0.0
	for _, p := range prices {
		mean += p
	}
	mean /= float64(len(prices))

	variance := 0.0
	for _, p := range prices {
		variance += (p - mean) * (p - mean)
	}
	variance /= float64(len(prices))
	stdDev := 0.0
	if variance > 0 {
		// Simple sqrt approximation
		for i := 0; i < 10; i++ {
			if stdDev == 0 {
				stdDev = variance
			} else {
				stdDev = (stdDev + variance/stdDev) / 2
			}
		}
	}

	// Return standard deviation as percentage points
	return stdDev
}

// GetPriceChange computes price change over a time window
func (ts *TimeSeriesStore) GetPriceChange(ticker string, window time.Duration) (float64, bool) {
	since := time.Now().Add(-window)
	snapshots := ts.GetSnapshots(ticker, since)

	if len(snapshots) < 2 {
		return 0, false
	}

	oldPrice := snapshots[0].MidPrice
	newPrice := snapshots[len(snapshots)-1].MidPrice

	return newPrice - oldPrice, true
}

