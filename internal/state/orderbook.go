package state

import (
	"sort"
	"strconv"
	"time"
)

type Orderbook struct {
	MarketTicker string       `json:"market_ticker"`
	Bids         []PriceLevel `json:"bids"` // Sorted descending by price
	Asks         []PriceLevel `json:"asks"` // Sorted ascending by price
	LastUpdate   time.Time    `json:"last_update"`
}

type PriceLevel struct {
	Price    int `json:"price"`    // In cents
	Quantity int `json:"quantity"`
}

func NewOrderbook(marketTicker string) *Orderbook {
	return &Orderbook{
		MarketTicker: marketTicker,
		Bids:         make([]PriceLevel, 0),
		Asks:         make([]PriceLevel, 0),
		LastUpdate:   time.Now(),
	}
}

func (ob *Orderbook) Clone() *Orderbook {
	bids := make([]PriceLevel, len(ob.Bids))
	copy(bids, ob.Bids)
	asks := make([]PriceLevel, len(ob.Asks))
	copy(asks, ob.Asks)
	return &Orderbook{
		MarketTicker: ob.MarketTicker,
		Bids:         bids,
		Asks:         asks,
		LastUpdate:   ob.LastUpdate,
	}
}

// UpdateFromKalshi updates the orderbook from Kalshi API response
// Kalshi returns yes_dollars and no_dollars arrays where each is [price_string, count_string]
// These are BIDS only. We synthesize ASKS:
// - YES asks = NO bids transformed: NO bid at X = YES ask at (100-X)
// - NO asks = YES bids transformed: YES bid at X = NO ask at (100-X)
// For a binary market, we track YES side and synthesize both YES and NO asks
func (ob *Orderbook) UpdateFromKalshi(resp *KalshiOrderbookResponse) {
	ob.Bids = ob.Bids[:0]
	ob.Asks = ob.Asks[:0]

	// YES bids become our YES bids
	for _, level := range resp.OrderbookFp.YesDollars {
		if len(level) < 2 {
			continue
		}
		priceCents, err1 := parseDollarToCents(level[0])
		qty, err2 := parseFixedPointCount(level[1])
		if err1 == nil && err2 == nil {
			ob.Bids = append(ob.Bids, PriceLevel{
				Price:    priceCents,
				Quantity: qty,
			})
		}
	}

	// NO bids become YES asks (synthesized): NO bid at X = YES ask at (100-X)
	for _, level := range resp.OrderbookFp.NoDollars {
		if len(level) < 2 {
			continue
		}
		noPriceCents, err1 := parseDollarToCents(level[0])
		qty, err2 := parseFixedPointCount(level[1])
		if err1 == nil && err2 == nil {
			// Convert NO bid price to YES ask: NO bid at X = YES ask at (100-X)
			yesAskPrice := 10000 - noPriceCents // 100.00 dollars = 10000 cents
			ob.Asks = append(ob.Asks, PriceLevel{
				Price:    yesAskPrice,
				Quantity: qty,
			})
		}
	}

	// Note: We synthesize YES asks from NO bids above.
	// For a binary market, tracking YES side is sufficient.
	// NO side can be derived: NO price = 100 - YES price

	// Sort bids descending (best bid first), asks ascending (best ask first)
	sort.Slice(ob.Bids, func(i, j int) bool {
		return ob.Bids[i].Price > ob.Bids[j].Price
	})
	sort.Slice(ob.Asks, func(i, j int) bool {
		return ob.Asks[i].Price < ob.Asks[j].Price
	})

	ob.LastUpdate = time.Now()
}

func (ob *Orderbook) Spread() (int, bool) {
	if len(ob.Bids) == 0 || len(ob.Asks) == 0 {
		return 0, false
	}
	bestBid := ob.Bids[0].Price
	bestAsk := ob.Asks[0].Price
	return bestAsk - bestBid, true
}

func (ob *Orderbook) BidDepth() int64 {
	var depth int64
	for _, level := range ob.Bids {
		depth += int64(level.Price) * int64(level.Quantity)
	}
	return depth
}

func (ob *Orderbook) AskDepth() int64 {
	var depth int64
	for _, level := range ob.Asks {
		depth += int64(level.Price) * int64(level.Quantity)
	}
	return depth
}

func (ob *Orderbook) ImbalanceRatio() float64 {
	bidDepth := float64(ob.BidDepth())
	askDepth := float64(ob.AskDepth())
	total := bidDepth + askDepth
	if total == 0 {
		return 0.0
	}
	return (bidDepth - askDepth) / total
}

// Microprice computes volume-weighted mid price (microprice)
// This is a better estimate of fair value than simple mid
func (ob *Orderbook) Microprice() (float64, bool) {
	if len(ob.Bids) == 0 || len(ob.Asks) == 0 {
		return 0, false
	}

	bestBid := float64(ob.Bids[0].Price) / 100.0
	bestAsk := float64(ob.Asks[0].Price) / 100.0
	bidQty := float64(ob.Bids[0].Quantity)
	askQty := float64(ob.Asks[0].Quantity)

	// Volume-weighted mid: (bid*askQty + ask*bidQty) / (bidQty + askQty)
	totalQty := bidQty + askQty
	if totalQty == 0 {
		return (bestBid + bestAsk) / 2.0, true
	}

	microprice := (bestBid*askQty + bestAsk*bidQty) / totalQty
	return microprice, true
}

// DepthAtPrice returns total quantity available within X cents of mid
func (ob *Orderbook) DepthAtPrice(centsFromMid int) (int64, int64) {
	if len(ob.Bids) == 0 || len(ob.Asks) == 0 {
		return 0, 0
	}

	bestBid := int64(ob.Bids[0].Price)
	bestAsk := int64(ob.Asks[0].Price)
	mid := (bestBid + bestAsk) / 2

	var bidDepth, askDepth int64
	bidThreshold := mid - int64(centsFromMid)
	askThreshold := mid + int64(centsFromMid)

	for _, level := range ob.Bids {
		if int64(level.Price) >= bidThreshold {
			bidDepth += int64(level.Quantity)
		}
	}

	for _, level := range ob.Asks {
		if int64(level.Price) <= askThreshold {
			askDepth += int64(level.Quantity)
		}
	}

	return bidDepth, askDepth
}

// KalshiOrderbookResponse represents the API response structure
type KalshiOrderbookResponse struct {
	OrderbookFp KalshiOrderbookFp `json:"orderbook_fp"`
}

type KalshiOrderbookFp struct {
	YesDollars [][]string `json:"yes_dollars"`
	NoDollars  [][]string `json:"no_dollars"`
}

func parseDollarToCents(s string) (int, error) {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	return int(f * 100), nil
}

func parseFixedPointCount(s string) (int, error) {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	return int(f), nil
}

