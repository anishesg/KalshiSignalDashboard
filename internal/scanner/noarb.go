package scanner

import (
	"fmt"
	"time"

	"github.com/kalshi-signal-feed/internal/state"
)

// EventGroup represents a set of mutually exclusive markets
type EventGroup struct {
	EventTicker string   `json:"event_ticker"`
	Markets     []string `json:"markets"` // market tickers
}

// NoArbViolation represents a detected arbitrage opportunity
type NoArbViolation struct {
	EventTicker      string    `json:"event_ticker"`
	Markets          []string  `json:"markets"`
	SumBuyPrice      float64   `json:"sum_buy_price"`      // cost to buy all outcomes
	SumSellPrice     float64   `json:"sum_sell_price"`     // revenue from selling all outcomes
	NetArb           float64   `json:"net_arb"`            // net profit (after fees)
	EstimatedFees    float64   `json:"estimated_fees"`      // estimated fees
	EstimatedSlippage float64   `json:"estimated_slippage"` // estimated slippage
	Liquidity        int64     `json:"liquidity"`           // min available size
	Timestamp        time.Time `json:"timestamp"`
	Actionable       bool      `json:"actionable"`          // true if net_arb > threshold
}

// NoArbEngine detects cross-market arbitrage opportunities
type NoArbEngine struct {
	state *state.Engine
}

func NewNoArbEngine(stateEngine *state.Engine) *NoArbEngine {
	return &NoArbEngine{
		state: stateEngine,
	}
}

// GroupMarketsByEvent groups markets by event_ticker
func (n *NoArbEngine) GroupMarketsByEvent() map[string][]string {
	markets := n.state.GetAllMarkets()
	groups := make(map[string][]string)

	for _, market := range markets {
		if market.Status != state.StatusActive {
			continue
		}
		eventTicker := market.EventTicker
		if eventTicker == "" {
			continue
		}
		groups[eventTicker] = append(groups[eventTicker], market.Ticker)
	}

	return groups
}

// CheckNoArbViolations checks for arbitrage opportunities within event groups
func (n *NoArbEngine) CheckNoArbViolations() []NoArbViolation {
	groups := n.GroupMarketsByEvent()
	var violations []NoArbViolation

	for eventTicker, marketTickers := range groups {
		if len(marketTickers) < 2 {
			continue // Need at least 2 markets for arbitrage
		}

		violation := n.checkEventGroup(eventTicker, marketTickers)
		if violation != nil {
			violations = append(violations, *violation)
		}
	}

	return violations
}

func (n *NoArbEngine) checkEventGroup(eventTicker string, marketTickers []string) *NoArbViolation {
	var sumBuyPrice float64  // Cost to buy all outcomes (best ask prices)
	var sumSellPrice float64 // Revenue from selling all outcomes (best bid prices)
	var minLiquidity int64 = 1000000 // Start high, find minimum

	allMarketsValid := true

	for _, ticker := range marketTickers {
		orderbook, exists := n.state.GetOrderbook(ticker)
		if !exists || len(orderbook.Bids) == 0 || len(orderbook.Asks) == 0 {
			allMarketsValid = false
			break
		}

		// Best ask = cost to buy YES
		bestAsk := float64(orderbook.Asks[0].Price) / 100.0 // Convert cents to probability
		sumBuyPrice += bestAsk

		// Best bid = revenue from selling YES
		bestBid := float64(orderbook.Bids[0].Price) / 100.0
		sumSellPrice += bestBid

		// Track minimum available liquidity
		bidDepth := int64(orderbook.Bids[0].Quantity)
		askDepth := int64(orderbook.Asks[0].Quantity)
		if bidDepth < minLiquidity {
			minLiquidity = bidDepth
		}
		if askDepth < minLiquidity {
			minLiquidity = askDepth
		}
	}

	if !allMarketsValid {
		return nil
	}

	// In a perfect market, sum of all outcomes should = 100%
	// If sumBuyPrice < 100%, we can buy all outcomes for less than $1 (arbitrage)
	// If sumSellPrice > 100%, we can sell all outcomes for more than $1 (arbitrage)

	// Calculate net arbitrage
	// Buy arbitrage: if sumBuyPrice < 1.0, profit = 1.0 - sumBuyPrice
	// Sell arbitrage: if sumSellPrice > 1.0, profit = sumSellPrice - 1.0
	var netArb float64
	if sumBuyPrice < 1.0 {
		netArb = 1.0 - sumBuyPrice // Buy all outcomes, guaranteed $1 payout
	} else if sumSellPrice > 1.0 {
		netArb = sumSellPrice - 1.0 // Sell all outcomes, guaranteed $1 cost
	} else {
		return nil // No arbitrage
	}

	// Estimate fees (Kalshi typically charges ~5-10% on trades)
	// For simplicity, assume 5% on each leg
	estimatedFees := sumBuyPrice * 0.05 * float64(len(marketTickers))
	if sumSellPrice > 1.0 {
		estimatedFees = sumSellPrice * 0.05 * float64(len(marketTickers))
	}

	// Estimate slippage (walking the book)
	estimatedSlippage := 0.01 * float64(len(marketTickers)) // 1% per market

	// Net arbitrage after fees and slippage
	netArbAfterCosts := netArb - estimatedFees - estimatedSlippage

	// Only flag if net arbitrage exceeds threshold (e.g., 2 cents)
	actionable := netArbAfterCosts > 0.02 && minLiquidity >= 10

	violation := &NoArbViolation{
		EventTicker:       eventTicker,
		Markets:           marketTickers,
		SumBuyPrice:       sumBuyPrice,
		SumSellPrice:      sumSellPrice,
		NetArb:            netArbAfterCosts,
		EstimatedFees:     estimatedFees,
		EstimatedSlippage: estimatedSlippage,
		Liquidity:         minLiquidity,
		Timestamp:         time.Now(),
		Actionable:        actionable,
	}

	return violation
}

// FormatViolation returns a human-readable description
func (v *NoArbViolation) FormatViolation() string {
	if v.SumBuyPrice < 1.0 {
		return fmt.Sprintf(
			"BUY ARB: Event %s - Buy all outcomes for %.2f¢, guaranteed $1 payout. Net after costs: %.2f¢. Liquidity: %d contracts",
			v.EventTicker,
			v.SumBuyPrice*100,
			v.NetArb*100,
			v.Liquidity,
		)
	} else {
		return fmt.Sprintf(
			"SELL ARB: Event %s - Sell all outcomes for %.2f¢, guaranteed $1 cost. Net after costs: %.2f¢. Liquidity: %d contracts",
			v.EventTicker,
			v.SumSellPrice*100,
			v.NetArb*100,
			v.Liquidity,
		)
	}
}

