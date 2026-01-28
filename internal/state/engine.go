package state

import (
	"sync"
	"time"
)

type Engine struct {
	mu         sync.RWMutex
	markets    map[string]*Market
	orderbooks map[string]*Orderbook
	tradeLogs  map[string]*TradeLog
	timeSeries *TimeSeriesStore
}

func NewEngine() *Engine {
	return &Engine{
		markets:    make(map[string]*Market),
		orderbooks: make(map[string]*Orderbook),
		tradeLogs:  make(map[string]*TradeLog),
		timeSeries: NewTimeSeriesStore(),
	}
}

func (e *Engine) RegisterMarket(market *Market) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.markets[market.Ticker] = market
	if _, exists := e.orderbooks[market.Ticker]; !exists {
		e.orderbooks[market.Ticker] = NewOrderbook(market.Ticker)
	}
}

func (e *Engine) UpdateOrderbook(ticker string, orderbook *Orderbook) {
	e.mu.Lock()
	e.orderbooks[ticker] = orderbook
	e.mu.Unlock()

	// Record snapshot for time-series (call GetRecentTrades after releasing lock to avoid deadlock)
	trades := e.GetRecentTrades(ticker, 5*time.Minute)
	e.timeSeries.RecordSnapshot(ticker, orderbook, trades)
}

func (e *Engine) AddTrade(trade *Trade) {
	e.mu.Lock()
	defer e.mu.Unlock()

	log, exists := e.tradeLogs[trade.MarketTicker]
	if !exists {
		log = NewTradeLog()
		e.tradeLogs[trade.MarketTicker] = log
	}
	log.Add(trade)

	// Record trade in time-series
	e.timeSeries.RecordTrade(trade.MarketTicker, trade)
}

func (e *Engine) GetOrderbook(ticker string) (*Orderbook, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	ob, exists := e.orderbooks[ticker]
	if !exists {
		return nil, false
	}
	return ob.Clone(), true
}

func (e *Engine) GetMarket(ticker string) (*Market, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	m, exists := e.markets[ticker]
	if !exists {
		return nil, false
	}
	return m.Clone(), true
}

func (e *Engine) GetAllMarkets() []*Market {
	e.mu.RLock()
	defer e.mu.RUnlock()

	markets := make([]*Market, 0, len(e.markets))
	for _, m := range e.markets {
		markets = append(markets, m.Clone())
	}
	return markets
}

func (e *Engine) GetRecentTrades(ticker string, window time.Duration) []*Trade {
	e.mu.RLock()
	defer e.mu.RUnlock()

	log, exists := e.tradeLogs[ticker]
	if !exists {
		return nil
	}

	cutoff := time.Now().Add(-window)
	return log.GetSince(cutoff)
}

func (e *Engine) GetTimeSeries() *TimeSeriesStore {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.timeSeries
}

