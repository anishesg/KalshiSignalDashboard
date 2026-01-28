package signals

import "time"

type SignalType string

const (
	SignalTypeImpliedProbabilityDrift SignalType = "implied_probability_drift"
	SignalTypeOrderbookImbalance      SignalType = "orderbook_imbalance"
	SignalTypeVolumeSurge             SignalType = "volume_surge"
)

type Signal struct {
	MarketTicker string    `json:"market_ticker"`
	Type         SignalType `json:"type"`
	Value        float64   `json:"value"`
	Timestamp    time.Time `json:"timestamp"`
	Metadata     SignalMetadata `json:"metadata"`

	// Type-specific data (only one will be set)
	ImpliedProbabilityDrift *ImpliedProbabilityDriftData `json:"implied_probability_drift,omitempty"`
	OrderbookImbalance      *OrderbookImbalanceData      `json:"orderbook_imbalance,omitempty"`
	VolumeSurge             *VolumeSurgeData             `json:"volume_surge,omitempty"`
}

type SignalMetadata struct {
	PreviousValue    *float64 `json:"previous_value,omitempty"`
	ThresholdCrossed bool     `json:"threshold_crossed"`
	Confidence       float64  `json:"confidence"` // 0.0 to 1.0
}

type ImpliedProbabilityDriftData struct {
	Delta      float64 `json:"delta"`
	WindowSecs int     `json:"window_secs"`
}

type OrderbookImbalanceData struct {
	BidRatio   float64 `json:"bid_ratio"`
	SpreadCents int    `json:"spread_cents"`
}

type VolumeSurgeData struct {
	VolumeMultiplier float64 `json:"volume_multiplier"`
	WindowSecs       int     `json:"window_secs"`
}

