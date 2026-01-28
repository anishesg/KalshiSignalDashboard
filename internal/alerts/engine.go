package alerts

import (
	"time"

	"github.com/kalshi-signal-feed/internal/scanner"
	"github.com/kalshi-signal-feed/internal/state"
)

// AlertType represents the type of alert
type AlertType string

const (
	AlertTypeSpreadTightened    AlertType = "spread_tightened"
	AlertTypeDepthIncreased     AlertType = "depth_increased"
	AlertTypeImbalancePressure  AlertType = "imbalance_pressure"
	AlertTypeNoArbViolation     AlertType = "no_arb_violation"
	AlertTypeExecutionReady     AlertType = "execution_ready"
	AlertTypePriceDrift         AlertType = "price_drift"
)

// Alert represents a mechanical trading alert
type Alert struct {
	ID            string                 `json:"id"`
	Type          AlertType              `json:"type"`
	MarketTicker  string                 `json:"market_ticker"`
	Title         string                 `json:"title"`
	Timestamp     time.Time              `json:"timestamp"`
	
	// Why it fired
	Reason        string                 `json:"reason"`
	Inputs        map[string]interface{} `json:"inputs"`
	Threshold     float64                `json:"threshold"`
	CurrentValue  float64                `json:"current_value"`
	
	// What it suggests
	Suggestion    string                 `json:"suggestion"`
	Action        string                 `json:"action"` // "buy", "sell", "watch", "skip"
	
	// Confidence (from backtesting)
	Confidence    float64                `json:"confidence"`     // 0-1
	HitRate       float64                `json:"hit_rate"`       // historical hit rate
	SampleSize    int                    `json:"sample_size"`     // number of historical samples
	
	// Execution context
	EstimatedEdge    float64 `json:"estimated_edge"`     // cents
	EstimatedSlippage float64 `json:"estimated_slippage"` // cents
	CanExecute       bool    `json:"can_execute"`
	RecommendedSize  int     `json:"recommended_size"`   // contracts
	
	// Risk context
	TimeToExpiry   float64 `json:"time_to_expiry"` // hours
	CurrentExposure float64 `json:"current_exposure"` // if tracking positions
}

// Engine generates mechanical alerts based on market conditions
type Engine struct {
	state        *state.Engine
	scanner      *scanner.Scanner
	noArbEngine  *scanner.NoArbEngine
	backtest     *BacktestHarness
	alertHistory map[string][]Alert // market_ticker -> alerts
}

func NewEngine(stateEngine *state.Engine) *Engine {
	scan := scanner.NewScanner(stateEngine)
	noArbEngine := scanner.NewNoArbEngine(stateEngine)
	backtest := NewBacktestHarness(stateEngine)
	
	return &Engine{
		state:        stateEngine,
		scanner:      scan,
		noArbEngine:  noArbEngine,
		backtest:     backtest,
		alertHistory: make(map[string][]Alert),
	}
}

// CheckAlerts scans markets and generates alerts
func (e *Engine) CheckAlerts() []Alert {
	var alerts []Alert
	
	// Check all opportunities
	opportunities := e.scanner.ScanMarkets()
	
	for _, opp := range opportunities {
		marketAlerts := e.checkMarketAlerts(opp)
		alerts = append(alerts, marketAlerts...)
	}
	
	// Check no-arb violations
	violations := e.noArbEngine.CheckNoArbViolations()
	for _, violation := range violations {
		if violation.Actionable {
			alert := e.createNoArbAlert(violation)
			alerts = append(alerts, alert)
		}
	}
	
	// Store in history
	for _, alert := range alerts {
		e.alertHistory[alert.MarketTicker] = append(e.alertHistory[alert.MarketTicker], alert)
	}
	
	return alerts
}

func (e *Engine) checkMarketAlerts(opp scanner.MarketOpportunity) []Alert {
	var alerts []Alert
	
	// 1. Spread tightened
	if opp.SpreadPercent < 0.5 && opp.SpreadPercent > 0 { // Spread < 0.5%
		alert := Alert{
			ID:           generateAlertID(opp.MarketTicker, AlertTypeSpreadTightened),
			Type:         AlertTypeSpreadTightened,
			MarketTicker: opp.MarketTicker,
			Title:        opp.Title,
			Timestamp:    time.Now(),
			Reason:       "Spread tightened below 0.5%",
			Inputs: map[string]interface{}{
				"spread_percent": opp.SpreadPercent,
			},
			Threshold:    0.5,
			CurrentValue: opp.SpreadPercent,
			Suggestion:   "Liquidity improved: easier to enter/exit",
			Action:       "watch",
			CanExecute:   opp.CanExecute100,
			EstimatedSlippage: float64(opp.EstimatedSlippage100) / 100.0,
		}
		
		// Get confidence from backtest
		confidence, hitRate, sampleSize := e.backtest.GetAlertStats(opp.MarketTicker, AlertTypeSpreadTightened)
		alert.Confidence = confidence
		alert.HitRate = hitRate
		alert.SampleSize = sampleSize
		
		alerts = append(alerts, alert)
	}
	
	// 2. Depth increased
	if opp.DepthAtTop5 > 500 { // >500 contracts at top 5 levels
		alert := Alert{
			ID:           generateAlertID(opp.MarketTicker, AlertTypeDepthIncreased),
			Type:         AlertTypeDepthIncreased,
			MarketTicker: opp.MarketTicker,
			Title:        opp.Title,
			Timestamp:    time.Now(),
			Reason:       "Depth at top-5 levels exceeds 500 contracts",
			Inputs: map[string]interface{}{
				"depth_at_top5": opp.DepthAtTop5,
			},
			Threshold:    500,
			CurrentValue:  float64(opp.DepthAtTop5),
			Suggestion:    "High liquidity: can execute larger size",
			Action:        "watch",
			CanExecute:    true,
			RecommendedSize: int(opp.DepthAtTop5 / 2), // Conservative
		}
		
		confidence, hitRate, sampleSize := e.backtest.GetAlertStats(opp.MarketTicker, AlertTypeDepthIncreased)
		alert.Confidence = confidence
		alert.HitRate = hitRate
		alert.SampleSize = sampleSize
		
		alerts = append(alerts, alert)
	}
	
	// 3. Imbalance pressure (imbalance high but price hasn't moved)
	if absFloat(opp.Imbalance) > 0.6 && absFloat(opp.MicropriceDiff) > 1.0 {
		direction := "buy"
		if opp.Imbalance < 0 {
			direction = "sell"
		}
		
		alert := Alert{
			ID:           generateAlertID(opp.MarketTicker, AlertTypeImbalancePressure),
			Type:         AlertTypeImbalancePressure,
			MarketTicker: opp.MarketTicker,
			Title:        opp.Title,
			Timestamp:    time.Now(),
			Reason:       "Strong orderbook imbalance detected with price lag",
			Inputs: map[string]interface{}{
				"imbalance":      opp.Imbalance,
				"microprice_diff": opp.MicropriceDiff,
			},
			Threshold:    0.6,
			CurrentValue: absFloat(opp.Imbalance),
			Suggestion:   "Pressure detected: watch for price movement",
			Action:       direction,
			CanExecute:   opp.CanExecute100,
			EstimatedSlippage: float64(opp.EstimatedSlippage100) / 100.0,
		}
		
		confidence, hitRate, sampleSize := e.backtest.GetAlertStats(opp.MarketTicker, AlertTypeImbalancePressure)
		alert.Confidence = confidence
		alert.HitRate = hitRate
		alert.SampleSize = sampleSize
		
		alerts = append(alerts, alert)
	}
	
	// 4. Execution ready (good liquidity + tight spread)
	if opp.LiquidityScore > 0.7 && opp.SpreadPercent < 1.0 && opp.CanExecute100 {
		alert := Alert{
			ID:           generateAlertID(opp.MarketTicker, AlertTypeExecutionReady),
			Type:         AlertTypeExecutionReady,
			MarketTicker: opp.MarketTicker,
			Title:        opp.Title,
			Timestamp:    time.Now(),
			Reason:       "Optimal execution conditions: tight spread + good depth",
			Inputs: map[string]interface{}{
				"liquidity_score": opp.LiquidityScore,
				"spread_percent":  opp.SpreadPercent,
			},
			Threshold:    0.7,
			CurrentValue:  opp.LiquidityScore,
			Suggestion:    "Good entry/exit conditions",
			Action:        "watch",
			CanExecute:    true,
			EstimatedSlippage: float64(opp.EstimatedSlippage100) / 100.0,
			RecommendedSize: 100,
		}
		
		confidence, hitRate, sampleSize := e.backtest.GetAlertStats(opp.MarketTicker, AlertTypeExecutionReady)
		alert.Confidence = confidence
		alert.HitRate = hitRate
		alert.SampleSize = sampleSize
		
		alerts = append(alerts, alert)
	}
	
	return alerts
}

func (e *Engine) createNoArbAlert(violation scanner.NoArbViolation) Alert {
	alert := Alert{
		ID:           generateAlertID(violation.EventTicker, AlertTypeNoArbViolation),
		Type:         AlertTypeNoArbViolation,
		MarketTicker: violation.EventTicker,
		Title:        violation.FormatViolation(),
		Timestamp:    time.Now(),
		Reason:       "Arbitrage opportunity detected",
		Inputs: map[string]interface{}{
			"sum_buy_price":  violation.SumBuyPrice,
			"sum_sell_price": violation.SumSellPrice,
			"net_arb":        violation.NetArb,
		},
		Threshold:    0.02,
		CurrentValue: violation.NetArb,
		Suggestion:   "Systematic arbitrage: execute if liquidity sufficient",
		Action:       "buy", // or "sell" depending on arb type
		CanExecute:   violation.Liquidity >= 10,
		EstimatedEdge: violation.NetArb * 100, // cents
		EstimatedSlippage: violation.EstimatedSlippage * 100,
		RecommendedSize: int(violation.Liquidity),
	}
	
	confidence, hitRate, sampleSize := e.backtest.GetAlertStats(violation.EventTicker, AlertTypeNoArbViolation)
	alert.Confidence = confidence
	alert.HitRate = hitRate
	alert.SampleSize = sampleSize
	
	return alert
}

func generateAlertID(marketTicker string, alertType AlertType) string {
	return marketTicker + "_" + string(alertType) + "_" + time.Now().Format("20060102150405")
}

func absFloat(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

