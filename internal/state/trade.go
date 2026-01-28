package state

import (
	"sync"
	"time"
)

type TradeSide string

const (
	SideYes TradeSide = "yes"
	SideNo  TradeSide = "no"
)

type Trade struct {
	MarketTicker string
	Side         TradeSide
	Price        int // In cents
	Quantity     int
	Timestamp    time.Time
}

type TradeLog struct {
	mu     sync.RWMutex
	trades []*Trade
	maxLen int
}

func NewTradeLog() *TradeLog {
	return &TradeLog{
		trades: make([]*Trade, 0, 1000),
		maxLen: 1000,
	}
}

func (tl *TradeLog) Add(trade *Trade) {
	tl.mu.Lock()
	defer tl.mu.Unlock()

	tl.trades = append(tl.trades, trade)

	// Keep only last maxLen trades
	if len(tl.trades) > tl.maxLen {
		tl.trades = tl.trades[len(tl.trades)-tl.maxLen:]
	}
}

func (tl *TradeLog) GetSince(cutoff time.Time) []*Trade {
	tl.mu.RLock()
	defer tl.mu.RUnlock()

	var result []*Trade
	for _, trade := range tl.trades {
		if trade.Timestamp.After(cutoff) || trade.Timestamp.Equal(cutoff) {
			result = append(result, trade)
		}
	}
	return result
}

