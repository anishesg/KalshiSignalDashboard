package signals

import (
	"context"
	"time"

	"github.com/kalshi-signal-feed/internal/config"
	"github.com/kalshi-signal-feed/internal/state"
)

type Processor struct {
	state      *state.Engine
	signalChan chan<- Signal
	config     config.SignalConfig
}

func NewProcessor(state *state.Engine, signalChan chan<- Signal, cfg config.SignalConfig) *Processor {
	return &Processor{
		state:      state,
		signalChan: signalChan,
		config:     cfg,
	}
}

func (p *Processor) Run(ctx context.Context) error {
	ticker := time.NewTicker(time.Duration(p.config.ComputationIntervalSecs) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			p.computeSignals()
		}
	}
}

func (p *Processor) computeSignals() {
	markets := p.state.GetAllMarkets()

	for _, market := range markets {
		if market.Status != state.StatusActive {
			continue
		}

		orderbook, exists := p.state.GetOrderbook(market.Ticker)
		if !exists {
			continue
		}

		// Compute orderbook imbalance
		if signal := p.computeOrderbookImbalance(market.Ticker, orderbook); signal != nil {
			select {
			case p.signalChan <- *signal:
			default:
				// Channel full, skip
			}
		}

		// Compute implied probability drift
		if signal := p.computeImpliedProbabilityDrift(market.Ticker, orderbook); signal != nil {
			select {
			case p.signalChan <- *signal:
			default:
				// Channel full, skip
			}
		}

		// Detect volume surge
		if signal := p.detectVolumeSurge(market.Ticker); signal != nil {
			select {
			case p.signalChan <- *signal:
			default:
				// Channel full, skip
			}
		}

		// Compute quantitative signals (always compute, even if not threshold-crossed)
		trades := p.state.GetRecentTrades(market.Ticker, 5*time.Minute)
		if quantSig := ComputeQuantitativeSignals(market.Ticker, orderbook, trades, market.ExpirationTime); quantSig != nil {
			// Convert to regular signal for output
			signal := &Signal{
				MarketTicker: market.Ticker,
				Type:         SignalTypeOrderbookImbalance, // Use as base type
				Value:        quantSig.LiquidityScore,
				Timestamp:    quantSig.Timestamp,
				Metadata: SignalMetadata{
					Confidence: quantSig.EfficiencyScore,
				},
			}
			select {
			case p.signalChan <- *signal:
			default:
			}
		}
	}
}

func (p *Processor) computeOrderbookImbalance(ticker string, orderbook *state.Orderbook) *Signal {
	imbalanceRatio := orderbook.ImbalanceRatio()
	spread, hasSpread := orderbook.Spread()

	if !hasSpread {
		return nil
	}

	thresholdCrossed := abs(imbalanceRatio) > p.config.ImbalanceThreshold

	if thresholdCrossed {
		return &Signal{
			MarketTicker: ticker,
			Type: SignalTypeOrderbookImbalance,
			Value: imbalanceRatio,
			Timestamp: time.Now(),
			Metadata: SignalMetadata{
				ThresholdCrossed: true,
				Confidence:       min(abs(imbalanceRatio)/p.config.ImbalanceThreshold, 1.0),
			},
			OrderbookImbalance: &OrderbookImbalanceData{
				BidRatio:  imbalanceRatio,
				SpreadCents: spread,
			},
		}
	}

	return nil
}

func (p *Processor) computeImpliedProbabilityDrift(ticker string, orderbook *state.Orderbook) *Signal {
	if len(orderbook.Bids) == 0 || len(orderbook.Asks) == 0 {
		return nil
	}

	bestBid := float64(orderbook.Bids[0].Price) / 100.0
	bestAsk := float64(orderbook.Asks[0].Price) / 100.0
	currentProb := (bestBid + bestAsk) / 2.0

	// Get recent trades
	window := time.Duration(p.config.DriftWindowSecs) * time.Second
	trades := p.state.GetRecentTrades(ticker, window)

	if len(trades) == 0 {
		return nil
	}

	// Compute average probability from trades
	var sumProb float64
	for _, trade := range trades {
		sumProb += float64(trade.Price) / 100.0
	}
	avgProb := sumProb / float64(len(trades))

	// Compute standard deviation
	var variance float64
	for _, trade := range trades {
		prob := float64(trade.Price) / 100.0
		variance += (prob - avgProb) * (prob - avgProb)
	}
	variance /= float64(len(trades))
	stdDev := sqrt(variance)

	if stdDev == 0 {
		return nil
	}

	drift := (currentProb - avgProb) / stdDev
	thresholdCrossed := abs(drift) > p.config.DriftThreshold

	if thresholdCrossed {
		return &Signal{
			MarketTicker: ticker,
			Type: SignalTypeImpliedProbabilityDrift,
			Value: drift,
			Timestamp: time.Now(),
			Metadata: SignalMetadata{
				PreviousValue:    &avgProb,
				ThresholdCrossed: true,
				Confidence:       min(abs(drift)/p.config.DriftThreshold, 1.0),
			},
			ImpliedProbabilityDrift: &ImpliedProbabilityDriftData{
				Delta:      currentProb - avgProb,
				WindowSecs: p.config.DriftWindowSecs,
			},
		}
	}

	return nil
}

func (p *Processor) detectVolumeSurge(ticker string) *Signal {
	window := time.Duration(p.config.VolumeWindowSecs) * time.Second
	recentTrades := p.state.GetRecentTrades(ticker, window)

	if len(recentTrades) == 0 {
		return nil
	}

	var recentVolume int
	for _, trade := range recentTrades {
		recentVolume += trade.Quantity
	}

	// Get baseline volume from longer window (5x the recent window)
	baselineWindow := time.Duration(p.config.VolumeWindowSecs*5) * time.Second
	baselineTrades := p.state.GetRecentTrades(ticker, baselineWindow)

	if len(baselineTrades) < 2 {
		return nil
	}

	var baselineVolume int
	for _, trade := range baselineTrades {
		baselineVolume += trade.Quantity
	}

	baselineAvg := float64(baselineVolume) / 5.0
	if baselineAvg == 0 {
		return nil
	}

	surgeRatio := float64(recentVolume) / baselineAvg
	thresholdCrossed := surgeRatio > p.config.VolumeSurgeThreshold

	if thresholdCrossed {
		return &Signal{
			MarketTicker: ticker,
			Type: SignalTypeVolumeSurge,
			Value: surgeRatio,
			Timestamp: time.Now(),
			Metadata: SignalMetadata{
				PreviousValue:    &baselineAvg,
				ThresholdCrossed: true,
				Confidence:       min(surgeRatio/p.config.VolumeSurgeThreshold, 1.0),
			},
			VolumeSurge: &VolumeSurgeData{
				VolumeMultiplier: surgeRatio,
				WindowSecs:       p.config.VolumeWindowSecs,
			},
		}
	}

	return nil
}

// Helper functions
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func sqrt(x float64) float64 {
	// Simple Newton's method approximation
	if x == 0 {
		return 0
	}
	guess := x
	for i := 0; i < 10; i++ {
		guess = (guess + x/guess) / 2
	}
	return guess
}

