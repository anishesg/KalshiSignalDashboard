package alerts

import (
	"math"
	"time"

	"github.com/kalshi-signal-feed/internal/state"
)

// BacktestHarness validates alerts against historical data
type BacktestHarness struct {
	state *state.Engine
	stats map[string]AlertStats // alert_type_market -> stats
}

type AlertStats struct {
	HitRate    float64
	SampleSize int
	AvgMove    float64 // average price move after alert
	Confidence float64 // derived from hit rate and sample size
}

func NewBacktestHarness(stateEngine *state.Engine) *BacktestHarness {
	return &BacktestHarness{
		state: stateEngine,
		stats: make(map[string]AlertStats),
	}
}

// GetAlertStats returns historical performance for an alert type
func (b *BacktestHarness) GetAlertStats(marketTicker string, alertType AlertType) (confidence, hitRate float64, sampleSize int) {
	key := string(alertType) + "_" + marketTicker
	
	stats, exists := b.stats[key]
	if !exists {
		// No historical data yet - return low confidence to indicate uncertainty
		// Use 0.3 (30%) instead of 0.5 to show we don't have enough data
		return 0.3, 0.0, 0
	}
	
	return stats.Confidence, stats.HitRate, stats.SampleSize
}

// BacktestAlert validates an alert against historical data
func (b *BacktestHarness) BacktestAlert(alert Alert, lookbackWindow time.Duration) AlertStats {
	ts := b.state.GetTimeSeries()
	
	// Get snapshots before and after alert time
	beforeTime := alert.Timestamp.Add(-lookbackWindow)
	afterTime := alert.Timestamp.Add(lookbackWindow)
	
	beforeSnapshots := ts.GetSnapshots(alert.MarketTicker, beforeTime)
	afterSnapshots := ts.GetSnapshots(alert.MarketTicker, afterTime)
	
	if len(beforeSnapshots) == 0 || len(afterSnapshots) == 0 {
		return AlertStats{
			HitRate:    0.0,
			SampleSize: 0,
			Confidence: 0.0,
		}
	}
	
	// Find snapshot closest to alert time
	var beforeSnapshot, afterSnapshot state.MarketSnapshot
	beforeSnapshot = beforeSnapshots[len(beforeSnapshots)-1]
	
	// Find snapshot after alert
	for _, snap := range afterSnapshots {
		if snap.Timestamp.After(alert.Timestamp) {
			afterSnapshot = snap
			break
		}
	}
	
	if afterSnapshot.MarketTicker == "" {
		return AlertStats{
			HitRate:    0.0,
			SampleSize: 0,
			Confidence: 0.0,
		}
	}
	
	// Calculate price move
	priceMove := afterSnapshot.MidPrice - beforeSnapshot.MidPrice
	
	// Determine if alert was "correct" based on type
	hit := false
	switch alert.Type {
	case AlertTypeImbalancePressure:
		// If imbalance suggests buy and price went up, it's a hit
		if alert.Action == "buy" && priceMove > 0.5 {
			hit = true
		} else if alert.Action == "sell" && priceMove < -0.5 {
			hit = true
		}
	case AlertTypeSpreadTightened, AlertTypeDepthIncreased, AlertTypeExecutionReady:
		// These are informational - consider a hit if price moved (any direction)
		if math.Abs(priceMove) > 0.1 {
			hit = true
		}
	case AlertTypeNoArbViolation:
		// No-arb should be profitable if executed
		hit = alert.EstimatedEdge > alert.EstimatedSlippage
	default:
		hit = math.Abs(priceMove) > 0.5
	}
	
	// Update stats
	key := string(alert.Type) + "_" + alert.MarketTicker
	stats, exists := b.stats[key]
	if !exists {
		stats = AlertStats{
			SampleSize: 0,
			HitRate:    0.0,
			AvgMove:    0.0,
		}
	}
	
	stats.SampleSize++
	if hit {
		stats.HitRate = (stats.HitRate*float64(stats.SampleSize-1) + 1.0) / float64(stats.SampleSize)
	} else {
		stats.HitRate = (stats.HitRate * float64(stats.SampleSize-1)) / float64(stats.SampleSize)
	}
	
	stats.AvgMove = (stats.AvgMove*float64(stats.SampleSize-1) + priceMove) / float64(stats.SampleSize)
	
	// Confidence = hit rate adjusted by sample size
	// More samples = higher confidence in hit rate
	if stats.SampleSize < 10 {
		stats.Confidence = stats.HitRate * 0.5 // Low confidence with few samples
	} else if stats.SampleSize < 50 {
		stats.Confidence = stats.HitRate * 0.75
	} else {
		stats.Confidence = stats.HitRate // High confidence with many samples
	}
	
	b.stats[key] = stats
	
	return stats
}

// RunBacktest runs backtest on all historical alerts
func (b *BacktestHarness) RunBacktest(alertHistory map[string][]Alert, lookbackWindow time.Duration) {
	for _, alerts := range alertHistory {
		for _, alert := range alerts {
			b.BacktestAlert(alert, lookbackWindow)
		}
	}
}


