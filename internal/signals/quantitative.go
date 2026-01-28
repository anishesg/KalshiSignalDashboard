package signals

import (
	"math"
	"time"

	"github.com/kalshi-signal-feed/internal/state"
)

// QuantitativeSignal represents advanced quantitative metrics
type QuantitativeSignal struct {
	MarketTicker string    `json:"market_ticker"`
	Timestamp    time.Time `json:"timestamp"`
	
	// Market Efficiency Metrics
	EfficiencyScore    float64 `json:"efficiency_score"`     // 0-1, how well prices reflect information
	PriceVolatility    float64 `json:"price_volatility"`      // Standard deviation of price changes
	InformationFlow    float64 `json:"information_flow"`      // Rate of information arrival
	
	// Probability Calibration
	CalibrationError   float64 `json:"calibration_error"`     // How well-calibrated the market is
	ExpectedValue      float64 `json:"expected_value"`        // Current implied probability
	HistoricalMean     float64 `json:"historical_mean"`       // Mean probability over window
	
	// Liquidity Metrics
	BidAskSpread       float64 `json:"bid_ask_spread"`        // Spread in cents
	LiquidityScore     float64 `json:"liquidity_score"`       // 0-1, depth and tightness
	MarketDepth        int64   `json:"market_depth"`          // Total depth
	
	// Event-Driven Signals
	TimeToEvent        float64 `json:"time_to_event"`        // Hours until expiration
	EventVolatility    float64 `json:"event_volatility"`      // Volatility near event
	PreEventSignal     bool    `json:"pre_event_signal"`     // Signal before major event
	
	// Statistical Metrics
	SharpeRatio        float64 `json:"sharpe_ratio"`          // Risk-adjusted return metric
	ZScore             float64 `json:"z_score"`               // Standard deviations from mean
	TrendStrength      float64 `json:"trend_strength"`        // 0-1, strength of trend
}

// ComputeQuantitativeSignals computes advanced quantitative metrics
func ComputeQuantitativeSignals(ticker string, orderbook *state.Orderbook, trades []*state.Trade, expirationTime *time.Time) *QuantitativeSignal {
	if orderbook == nil {
		return nil
	}

	sig := &QuantitativeSignal{
		MarketTicker: ticker,
		Timestamp:    time.Now(),
	}

	// Compute basic price metrics
	bestBid, hasBid := getBestBid(orderbook)
	bestAsk, hasAsk := getBestAsk(orderbook)
	
	if !hasBid || !hasAsk {
		return nil
	}

	midPrice := (bestBid + bestAsk) / 2.0
	spread := bestAsk - bestBid
	
	sig.BidAskSpread = spread
	sig.ExpectedValue = midPrice / 100.0 // Convert cents to probability
	
	// Market Depth
	sig.MarketDepth = orderbook.BidDepth() + orderbook.AskDepth()
	
	// Liquidity Score (0-1, based on depth and spread tightness)
	sig.LiquidityScore = computeLiquidityScore(orderbook, spread)
	
	// Price Volatility from trades
	if len(trades) > 1 {
		sig.PriceVolatility = computeVolatility(trades)
		sig.HistoricalMean = computeMean(trades)
		sig.ZScore = computeZScore(midPrice/100.0, sig.HistoricalMean, sig.PriceVolatility)
		sig.TrendStrength = computeTrendStrength(trades)
	}
	
	// Market Efficiency (how tight is spread relative to volatility)
	if sig.PriceVolatility > 0 {
		sig.EfficiencyScore = math.Min(1.0, (spread/100.0) / sig.PriceVolatility)
	} else {
		sig.EfficiencyScore = 1.0 // Perfect efficiency if no volatility
	}
	
	// Information Flow (rate of trades)
	if len(trades) > 0 {
		timeWindow := 5 * time.Minute
		recentTrades := filterRecentTrades(trades, timeWindow)
		sig.InformationFlow = float64(len(recentTrades)) / timeWindow.Minutes()
	}
	
	// Calibration Error (difference between current price and historical mean)
	if sig.HistoricalMean > 0 {
		sig.CalibrationError = math.Abs(sig.ExpectedValue - sig.HistoricalMean)
	}
	
	// Event-Driven Metrics
	if expirationTime != nil {
		timeToEvent := expirationTime.Sub(time.Now())
		sig.TimeToEvent = timeToEvent.Hours()
		
		// Higher volatility near event
		if timeToEvent < 24*time.Hour {
			sig.EventVolatility = sig.PriceVolatility * 1.5
			sig.PreEventSignal = true
		} else {
			sig.EventVolatility = sig.PriceVolatility
		}
	}
	
	// Sharpe-like ratio (return per unit of volatility)
	if sig.PriceVolatility > 0 {
		returnSignal := sig.ExpectedValue - sig.HistoricalMean
		sig.SharpeRatio = returnSignal / sig.PriceVolatility
	}
	
	return sig
}

// Helper functions
func getBestBid(ob *state.Orderbook) (float64, bool) {
	if len(ob.Bids) == 0 {
		return 0, false
	}
	return float64(ob.Bids[0].Price), true
}

func getBestAsk(ob *state.Orderbook) (float64, bool) {
	if len(ob.Asks) == 0 {
		return 0, false
	}
	return float64(ob.Asks[0].Price), true
}

func computeLiquidityScore(ob *state.Orderbook, spread float64) float64 {
	// Score based on depth and spread
	depth := float64(ob.BidDepth() + ob.AskDepth())
	
	// Normalize depth (assume 10000 is good depth)
	depthScore := math.Min(1.0, depth/10000.0)
	
	// Spread score (tighter is better, 1 cent spread = perfect)
	spreadScore := math.Max(0.0, 1.0 - (spread/100.0))
	
	// Combined score
	return (depthScore*0.6 + spreadScore*0.4)
}

func computeVolatility(trades []*state.Trade) float64 {
	if len(trades) < 2 {
		return 0
	}
	
	prices := make([]float64, len(trades))
	for i, t := range trades {
		prices[i] = float64(t.Price) / 100.0
	}
	
	mean := computeMean(trades)
	
	var variance float64
	for _, p := range prices {
		variance += (p - mean) * (p - mean)
	}
	variance /= float64(len(prices))
	
	return math.Sqrt(variance)
}

func computeMean(trades []*state.Trade) float64 {
	if len(trades) == 0 {
		return 0
	}
	
	var sum float64
	for _, t := range trades {
		sum += float64(t.Price) / 100.0
	}
	return sum / float64(len(trades))
}

func computeZScore(current, mean, stdDev float64) float64 {
	if stdDev == 0 {
		return 0
	}
	return (current - mean) / stdDev
}

func computeTrendStrength(trades []*state.Trade) float64 {
	if len(trades) < 3 {
		return 0
	}
	
	// Simple linear regression slope
	n := float64(len(trades))
	var sumX, sumY, sumXY, sumX2 float64
	
	for i, t := range trades {
		x := float64(i)
		y := float64(t.Price) / 100.0
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}
	
	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
	
	// Normalize to 0-1 (assume max slope of 0.1 per trade is strong trend)
	return math.Min(1.0, math.Abs(slope)*10.0)
}

func filterRecentTrades(trades []*state.Trade, window time.Duration) []*state.Trade {
	cutoff := time.Now().Add(-window)
	var recent []*state.Trade
	for _, t := range trades {
		if t.Timestamp.After(cutoff) {
			recent = append(recent, t)
		}
	}
	return recent
}

