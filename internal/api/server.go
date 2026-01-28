package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/kalshi-signal-feed/internal/config"
	"github.com/kalshi-signal-feed/internal/alerts"
	"github.com/kalshi-signal-feed/internal/scanner"
	"github.com/kalshi-signal-feed/internal/signals"
	"github.com/kalshi-signal-feed/internal/state"
	"github.com/rs/cors"
)

type Server struct {
	config     config.APIConfig
	state      *state.Engine
	signalChan <-chan signals.Signal
	server     *http.Server
	signals    []signals.Signal
	alerts     []alerts.Alert
	mu         sync.RWMutex
}

func NewServer(cfg config.APIConfig, stateEngine *state.Engine, signalChan <-chan signals.Signal) *Server {
	return &Server{
		config:     cfg,
		state:      stateEngine,
		signalChan: signalChan,
		signals:    make([]signals.Signal, 0, 1000),
	}
}

func (s *Server) Run(ctx context.Context) error {
	router := mux.NewRouter()

	// Setup CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   s.config.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
		MaxAge:           3600,
	})

	// API routes
	api := router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/markets", s.getMarkets).Methods("GET")
	api.HandleFunc("/markets/{ticker}", s.getMarket).Methods("GET")
	api.HandleFunc("/markets/{ticker}/orderbook", s.getOrderbook).Methods("GET")
	api.HandleFunc("/markets/{ticker}/debug", s.getMarketDebug).Methods("GET")
	api.HandleFunc("/scanner/opportunities", s.getOpportunities).Methods("GET")
	api.HandleFunc("/scanner/noarb", s.getNoArbViolations).Methods("GET")
	api.HandleFunc("/alerts", s.getAlerts).Methods("GET")
	api.HandleFunc("/signals", s.getSignals).Methods("GET")
	api.HandleFunc("/stream/signals", s.streamSignals).Methods("GET")
	api.HandleFunc("/categories", s.getCategories).Methods("GET")
	api.HandleFunc("/health", s.getHealth).Methods("GET")

	handler := c.Handler(router)

	s.server = &http.Server{
		Addr:    s.config.BindAddress,
		Handler: handler,
	}

	// Start signal collector
	go s.collectSignals(ctx)
	
	// Start alert checker
	go s.collectAlerts(ctx)

	fmt.Printf("API server starting on %s\n", s.config.BindAddress)

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

func (s *Server) collectSignals(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case signal := <-s.signalChan:
			s.mu.Lock()
			s.signals = append(s.signals, signal)
			// Keep only last 1000 signals
			if len(s.signals) > 1000 {
				s.signals = s.signals[len(s.signals)-1000:]
			}
			s.mu.Unlock()
		}
	}
}

func (s *Server) getMarkets(w http.ResponseWriter, r *http.Request) {
	markets := s.state.GetAllMarkets()

	response := struct {
		Markets []*state.Market `json:"markets"`
		Count   int             `json:"count"`
	}{
		Markets: markets,
		Count:   len(markets),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) getMarket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ticker := vars["ticker"]

	market, exists := s.state.GetMarket(ticker)
	if !exists {
		http.Error(w, "Market not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(market)
}

func (s *Server) getOrderbook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ticker := vars["ticker"]

	orderbook, exists := s.state.GetOrderbook(ticker)
	if !exists {
		http.Error(w, "Orderbook not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orderbook)
}

func (s *Server) getSignals(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	signalsCopy := make([]signals.Signal, len(s.signals))
	copy(signalsCopy, s.signals)
	s.mu.RUnlock()

	// Get query parameters
	marketTicker := r.URL.Query().Get("market_ticker")
	signalType := r.URL.Query().Get("type")
	limitStr := r.URL.Query().Get("limit")

	// Filter signals
	filtered := make([]signals.Signal, 0)
	for _, sig := range signalsCopy {
		if marketTicker != "" && sig.MarketTicker != marketTicker {
			continue
		}
		if signalType != "" && string(sig.Type) != signalType {
			continue
		}
		filtered = append(filtered, sig)
	}

	// Apply limit
	limit := len(filtered)
	if limitStr != "" {
		if l, err := parseInt(limitStr); err == nil && l > 0 {
			limit = l
			if limit > len(filtered) {
				limit = len(filtered)
			}
		}
	}

	if limit < len(filtered) {
		filtered = filtered[len(filtered)-limit:]
	}

	response := struct {
		Signals []signals.Signal `json:"signals"`
		Count   int              `json:"count"`
	}{
		Signals: filtered,
		Count:   len(filtered),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) streamSignals(w http.ResponseWriter, r *http.Request) {
	// Upgrade to WebSocket would go here
	// For now, return SSE (Server-Sent Events)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Send initial connection message
	fmt.Fprintf(w, "data: {\"type\":\"connected\"}\n\n")
	flusher.Flush()

	// Create a ticker to send periodic updates
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	lastCount := 0
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			s.mu.RLock()
			currentCount := len(s.signals)
			s.mu.RUnlock()

			if currentCount > lastCount {
				// Send new signals
				s.mu.RLock()
				newSignals := s.signals[lastCount:]
				s.mu.RUnlock()

				for _, sig := range newSignals {
					data, _ := json.Marshal(sig)
					fmt.Fprintf(w, "data: %s\n\n", string(data))
					flusher.Flush()
				}
				lastCount = currentCount
			}
		}
	}
}

func (s *Server) getMarketDebug(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ticker := vars["ticker"]

	market, exists := s.state.GetMarket(ticker)
	if !exists {
		http.Error(w, "Market not found", http.StatusNotFound)
		return
	}

	orderbook, hasOrderbook := s.state.GetOrderbook(ticker)
	trades := s.state.GetRecentTrades(ticker, 5*time.Minute)

	debug := struct {
		MarketTicker        string    `json:"market_ticker"`
		MarketStatus       string    `json:"market_status"`
		HasOrderbook       bool      `json:"has_orderbook"`
		OrderbookTimestamp *time.Time `json:"orderbook_timestamp,omitempty"`
		BidLevels           int       `json:"bid_levels"`
		AskLevels           int       `json:"ask_levels"`
		BestBid             *int      `json:"best_bid,omitempty"`
		BestAsk             *int      `json:"best_ask,omitempty"`
		Spread              *int      `json:"spread,omitempty"`
		Microprice          *float64  `json:"microprice,omitempty"`
		TradeCount          int       `json:"trade_count"`
		LastTradeTimestamp  *time.Time `json:"last_trade_timestamp,omitempty"`
		SignalCount         int       `json:"signal_count"`
		LastSignalTimestamp *time.Time `json:"last_signal_timestamp,omitempty"`
	}{
		MarketTicker:  ticker,
		MarketStatus:  string(market.Status),
		HasOrderbook:  hasOrderbook,
		BidLevels:     0,
		AskLevels:     0,
		TradeCount:    len(trades),
		SignalCount:   0,
	}

	if hasOrderbook {
		debug.OrderbookTimestamp = &orderbook.LastUpdate
		debug.BidLevels = len(orderbook.Bids)
		debug.AskLevels = len(orderbook.Asks)

		if len(orderbook.Bids) > 0 {
			bid := orderbook.Bids[0].Price
			debug.BestBid = &bid
		}
		if len(orderbook.Asks) > 0 {
			ask := orderbook.Asks[0].Price
			debug.BestAsk = &ask
		}

		if spread, ok := orderbook.Spread(); ok {
			spreadInt := int(spread)
			debug.Spread = &spreadInt
		}

		if microprice, ok := orderbook.Microprice(); ok {
			debug.Microprice = &microprice
		}
	}

	if len(trades) > 0 {
		lastTrade := trades[len(trades)-1]
		debug.LastTradeTimestamp = &lastTrade.Timestamp
	}

	// Count signals for this market
	s.mu.RLock()
	for _, sig := range s.signals {
		if sig.MarketTicker == ticker {
			debug.SignalCount++
			if debug.LastSignalTimestamp == nil || sig.Timestamp.After(*debug.LastSignalTimestamp) {
				debug.LastSignalTimestamp = &sig.Timestamp
			}
		}
	}
	s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(debug)
}

func (s *Server) getHealth(w http.ResponseWriter, r *http.Request) {
	response := struct {
		Status    string    `json:"status"`
		Timestamp time.Time `json:"timestamp"`
		Markets   int       `json:"markets"`
	}{
		Status:    "healthy",
		Timestamp: time.Now(),
		Markets:   len(s.state.GetAllMarkets()),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) getOpportunities(w http.ResponseWriter, r *http.Request) {
	scan := scanner.NewScanner(s.state)
	opportunities := scan.ScanMarkets()

	response := struct {
		Opportunities []scanner.MarketOpportunity `json:"opportunities"`
		Count         int                         `json:"count"`
		Timestamp     time.Time                   `json:"timestamp"`
	}{
		Opportunities: opportunities,
		Count:         len(opportunities),
		Timestamp:     time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) getNoArbViolations(w http.ResponseWriter, r *http.Request) {
	engine := scanner.NewNoArbEngine(s.state)
	violations := engine.CheckNoArbViolations()

	response := struct {
		Violations []scanner.NoArbViolation `json:"violations"`
		Count      int                      `json:"count"`
		Timestamp  time.Time                `json:"timestamp"`
	}{
		Violations: violations,
		Count:      len(violations),
		Timestamp:  time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) collectAlerts(ctx context.Context) {
	alertEngine := alerts.NewEngine(s.state)
	ticker := time.NewTicker(5 * time.Second) // Check every 5 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			newAlerts := alertEngine.CheckAlerts()
			if len(newAlerts) > 0 {
				s.mu.Lock()
				s.alerts = append(s.alerts, newAlerts...)
				// Keep only last 1000 alerts
				if len(s.alerts) > 1000 {
					s.alerts = s.alerts[len(s.alerts)-1000:]
				}
				s.mu.Unlock()
			}
		}
	}
}

func (s *Server) getAlerts(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	alertsCopy := make([]alerts.Alert, len(s.alerts))
	copy(alertsCopy, s.alerts)
	s.mu.RUnlock()

	// Get query parameters
	marketTicker := r.URL.Query().Get("market_ticker")
	alertType := r.URL.Query().Get("type")
	limitStr := r.URL.Query().Get("limit")

	// Filter alerts
	filtered := make([]alerts.Alert, 0)
	for _, alert := range alertsCopy {
		if marketTicker != "" && alert.MarketTicker != marketTicker {
			continue
		}
		if alertType != "" && string(alert.Type) != alertType {
			continue
		}
		filtered = append(filtered, alert)
	}

	// Apply limit
	limit := len(filtered)
	if limitStr != "" {
		if l, err := parseInt(limitStr); err == nil && l > 0 {
			limit = l
			if limit > len(filtered) {
				limit = len(filtered)
			}
		}
	}

	if limit < len(filtered) {
		filtered = filtered[len(filtered)-limit:]
	}

	response := struct {
		Alerts    []alerts.Alert `json:"alerts"`
		Count     int            `json:"count"`
		Timestamp time.Time      `json:"timestamp"`
	}{
		Alerts:    filtered,
		Count:     len(filtered),
		Timestamp: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// categorizeMarket uses keyword matching to categorize markets based on their title
func categorizeMarket(title, ticker string) string {
	titleLower := strings.ToLower(title)
	tickerLower := strings.ToLower(ticker)
	combined := titleLower + " " + tickerLower
	
	// Elections - Federal (check these first as they're most specific)
	if strings.Contains(combined, "senate") {
		if strings.Contains(combined, "primary") || strings.Contains(combined, "nominee") || strings.Contains(combined, "nomination") {
			return "Elections - Senate Primaries"
		}
		if strings.Contains(combined, "race") || strings.Contains(combined, "election") {
			return "Elections - Senate"
		}
		return "Elections - Senate"
	}
	
	if (strings.Contains(combined, "house") || strings.Contains(combined, "congress")) && 
		(strings.Contains(combined, "seat") || strings.Contains(combined, "race") || strings.Contains(combined, "win") || 
		 strings.Contains(combined, "democratic") || strings.Contains(combined, "republican")) {
		if strings.Contains(combined, "primary") {
			return "Elections - House Primaries"
		}
		return "Elections - House"
	}
	
	if strings.Contains(combined, "president") && (strings.Contains(combined, "election") || strings.Contains(combined, "nominee") || strings.Contains(combined, "nomination")) {
		return "Elections - President"
	}
	
	if strings.Contains(combined, "governor") || strings.Contains(combined, "governorship") {
		if strings.Contains(combined, "primary") || strings.Contains(combined, "nominee") {
			return "Elections - Governor Primaries"
		}
		return "Elections - Governor"
	}
	
	if strings.Contains(combined, "attorney general") || (strings.Contains(combined, "attorney") && strings.Contains(combined, "general") && strings.Contains(combined, "race")) {
		return "Elections - Attorney General"
	}
	if strings.Contains(combined, "attorney") && strings.Contains(combined, "race") {
		return "Elections - Attorney General"
	}
	
	// Appointments & Confirmations (check before other matches)
	if strings.Contains(combined, "confirm") || strings.Contains(combined, "confirmation") {
		if strings.Contains(combined, "supreme court") || strings.Contains(combined, "justice") || strings.Contains(combined, "scotus") {
			return "Appointments - Supreme Court"
		}
		if strings.Contains(combined, "cabinet") || (strings.Contains(combined, "secretary") && !strings.Contains(combined, "state department")) {
			return "Appointments - Cabinet"
		}
		if strings.Contains(combined, "attorney") || strings.Contains(combined, "us attorney") || strings.Contains(combined, "u.s. attorney") {
			return "Appointments - Attorneys"
		}
		if strings.Contains(combined, "judge") || strings.Contains(combined, "judicial") {
			return "Appointments - Judiciary"
		}
		return "Appointments - Other"
	}
	
	if strings.Contains(combined, "appoint") && !strings.Contains(combined, "disappoint") {
		if strings.Contains(combined, "supreme court") || strings.Contains(combined, "justice") {
			return "Appointments - Supreme Court"
		}
		if strings.Contains(combined, "cabinet") || strings.Contains(combined, "secretary") {
			return "Appointments - Cabinet"
		}
		return "Appointments - Other"
	}
	
	if strings.Contains(combined, "supreme court") || strings.Contains(combined, "scotus") {
		return "Appointments - Supreme Court"
	}
	
	if strings.Contains(combined, "cabinet") || (strings.Contains(combined, "secretary") && !strings.Contains(combined, "state department")) {
		return "Appointments - Cabinet"
	}
	
	// White House & Executive
	if strings.Contains(combined, "white house") && strings.Contains(combined, "visit") {
		return "White House - Visits"
	}
	if strings.Contains(combined, "visit") && (strings.Contains(combined, "white house") || strings.Contains(combined, "whvisit")) {
		return "White House - Visits"
	}
	if strings.Contains(combined, "trump") && (strings.Contains(combined, "endorse") || strings.Contains(combined, "endorsement")) {
		return "Elections - Endorsements"
	}
	if strings.Contains(combined, "presidential") && !strings.Contains(combined, "election") {
		return "Executive - Presidential"
	}
	if strings.Contains(combined, "mar-a-lago") {
		return "White House - Visits"
	}
	
	// Legislation
	if strings.Contains(combined, "bill") && (strings.Contains(combined, "pass") || strings.Contains(combined, "become law") || strings.Contains(combined, "law")) {
		return "Legislation - Bills & Laws"
	}
	if strings.Contains(combined, "legislation") || (strings.Contains(combined, "law") && strings.Contains(combined, "become")) {
		return "Legislation - Bills & Laws"
	}
	if strings.Contains(combined, "congress") && (strings.Contains(combined, "pass") || strings.Contains(combined, "vote") || strings.Contains(combined, "resolution")) {
		return "Legislation - Congressional Votes"
	}
	if strings.Contains(combined, "resolution") && strings.Contains(combined, "pass") {
		return "Legislation - Congressional Votes"
	}
	
	// International
	if strings.Contains(combined, "prime minister") || strings.Contains(combined, "parliament") || strings.Contains(combined, "parliamentary") {
		return "International - Foreign Leaders"
	}
	if strings.Contains(combined, "head of state") || strings.Contains(combined, "government") && 
		(strings.Contains(combined, "venezuela") || strings.Contains(combined, "czech") || strings.Contains(combined, "mexico") || 
		 strings.Contains(combined, "netherlands") || strings.Contains(combined, "hungary") || strings.Contains(combined, "armenia")) {
		return "International - Foreign Leaders"
	}
	if strings.Contains(combined, "nato") || strings.Contains(combined, "alliance") {
		return "International - Alliances"
	}
	if strings.Contains(combined, "taiwan") || strings.Contains(combined, "china") || strings.Contains(combined, "russia") || 
		strings.Contains(combined, "ukraine") || strings.Contains(combined, "israel") || strings.Contains(combined, "iran") ||
		strings.Contains(combined, "venezuela") || strings.Contains(combined, "czech") || strings.Contains(combined, "mexico") ||
		strings.Contains(combined, "netherlands") || strings.Contains(combined, "hungary") || strings.Contains(combined, "armenia") ||
		strings.Contains(combined, "norway") || strings.Contains(combined, "philippines") || strings.Contains(combined, "chile") ||
		strings.Contains(combined, "paraguay") || strings.Contains(combined, "france") || strings.Contains(combined, "lyon") {
		return "International - Foreign Policy"
	}
	if strings.Contains(combined, "visit") && (strings.Contains(combined, "country") || strings.Contains(combined, "nation") || strings.Contains(combined, "foreign")) {
		return "International - Visits"
	}
	
	// Local Elections
	if strings.Contains(combined, "mayor") || strings.Contains(combined, "mayoral") {
		return "Elections - Local"
	}
	if strings.Contains(combined, "primary") && (strings.Contains(combined, "wa-") || strings.Contains(combined, "ca-") || 
		strings.Contains(combined, "tx-") || strings.Contains(combined, "ny-") || strings.Contains(combined, "fl-") ||
		strings.Contains(combined, "il-") || strings.Contains(combined, "mi-") || strings.Contains(combined, "nc-") ||
		strings.Contains(combined, "md-") || strings.Contains(combined, "az-") || strings.Contains(combined, "ga-")) {
		return "Elections - House Primaries"
	}
	
	// Economics
	if strings.Contains(combined, "gdp") || strings.Contains(combined, "inflation") || strings.Contains(combined, "unemployment") || 
		strings.Contains(combined, "recession") || strings.Contains(combined, "economic") {
		return "Economics - Indicators"
	}
	if strings.Contains(combined, "fed") || strings.Contains(combined, "federal reserve") || strings.Contains(combined, "jerome powell") {
		return "Economics - Federal Reserve"
	}
	if strings.Contains(combined, "budget") || strings.Contains(combined, "spending") || strings.Contains(combined, "debt ceiling") {
		return "Economics - Budget"
	}
	
	// Approval & Polls
	if strings.Contains(combined, "approval") && (strings.Contains(combined, "rating") || strings.Contains(combined, "below") || strings.Contains(combined, "above")) {
		return "Polls - Approval Ratings"
	}
	if strings.Contains(combined, "poll") && !strings.Contains(combined, "polling place") {
		return "Polls - Other"
	}
	
	// Arrests & Charges
	if strings.Contains(combined, "arrest") || strings.Contains(combined, "charge") || strings.Contains(combined, "indict") || 
		strings.Contains(combined, "charged") || strings.Contains(combined, "indicted") {
		return "Legal - Arrests & Charges"
	}
	
	// Impeachment
	if strings.Contains(combined, "impeach") {
		return "Legal - Impeachment"
	}
	
	// Contempt & Legal Actions
	if strings.Contains(combined, "contempt") {
		return "Legal - Contempt"
	}
	
	// Elections - Other
	if strings.Contains(combined, "primary") && (strings.Contains(combined, "nominee") || strings.Contains(combined, "win") || strings.Contains(combined, "who will")) {
		return "Elections - Primaries"
	}
	if strings.Contains(combined, "nominee") && (strings.Contains(combined, "democratic") || strings.Contains(combined, "republican")) {
		return "Elections - Nominations"
	}
	if strings.Contains(combined, "election") && !strings.Contains(combined, "president") {
		if strings.Contains(combined, "foreign") || strings.Contains(combined, "international") {
			return "International - Foreign Leaders"
		}
		// Don't default to "Elections - Other" here, let it fall through to more specific checks
	}
	
	// Policy & Regulations
	if strings.Contains(combined, "policy") || strings.Contains(combined, "regulation") || strings.Contains(combined, "regulate") {
		return "Policy - Regulations"
	}
	if strings.Contains(combined, "executive order") || strings.Contains(combined, "order") && strings.Contains(combined, "come into effect") {
		return "Executive - Orders"
	}
	if strings.Contains(combined, "birthright") || strings.Contains(combined, "executive action") {
		return "Executive - Orders"
	}
	
	// Trade & Tariffs
	if strings.Contains(combined, "tariff") || strings.Contains(combined, "trade war") || strings.Contains(combined, "trade agreement") {
		return "Economics - Trade"
	}
	
	// Immigration
	if strings.Contains(combined, "immigration") || strings.Contains(combined, "border") || strings.Contains(combined, "deport") {
		return "Policy - Immigration"
	}
	
	// Healthcare
	if strings.Contains(combined, "healthcare") || strings.Contains(combined, "health care") || strings.Contains(combined, "medicare") || strings.Contains(combined, "medicaid") {
		return "Policy - Healthcare"
	}
	
	// Climate & Environment
	if strings.Contains(combined, "climate") || strings.Contains(combined, "carbon") || strings.Contains(combined, "emission") {
		return "Policy - Climate"
	}
	
	// Technology & Privacy
	if strings.Contains(combined, "privacy") || strings.Contains(combined, "data protection") || strings.Contains(combined, "tech regulation") {
		return "Policy - Technology"
	}
	
	// Capital Controls & Economic Policy
	if strings.Contains(combined, "capital control") {
		return "Economics - Policy"
	}
	
	// Medal & Awards
	if strings.Contains(combined, "medal of freedom") || strings.Contains(combined, "presidential medal") {
		return "Executive - Awards"
	}
	
	// Default to Misc
	return "Misc"
}

func (s *Server) getCategories(w http.ResponseWriter, r *http.Request) {
	markets := s.state.GetAllMarkets()
	
	// Group markets by intelligent categorization and event_ticker
	categoryMap := make(map[string]map[string][]*state.Market)
	
	for _, market := range markets {
		if market.Status != state.StatusActive {
			continue
		}
		
		// Use intelligent categorization based on title
		category := categorizeMarket(market.Title, market.Ticker)
		
		if categoryMap[category] == nil {
			categoryMap[category] = make(map[string][]*state.Market)
		}
		
		eventTicker := market.EventTicker
		if eventTicker == "" {
			eventTicker = "General"
		}
		
		categoryMap[category][eventTicker] = append(categoryMap[category][eventTicker], market)
	}
	
	// Build response structure
	type CategoryGroup struct {
		Category     string   `json:"category"`
		EventTickers []string `json:"event_tickers"`
		TotalMarkets int      `json:"total_markets"`
		Events       map[string]struct {
			EventTicker string         `json:"event_ticker"`
			Markets     []*state.Market `json:"markets"`
			Count       int             `json:"count"`
		} `json:"events"`
	}
	
	var categories []CategoryGroup
	for category, events := range categoryMap {
		eventList := make([]string, 0, len(events))
		eventDetails := make(map[string]struct {
			EventTicker string         `json:"event_ticker"`
			Markets     []*state.Market `json:"markets"`
			Count       int             `json:"count"`
		})
		
		totalMarkets := 0
		for eventTicker, markets := range events {
			eventList = append(eventList, eventTicker)
			eventDetails[eventTicker] = struct {
				EventTicker string         `json:"event_ticker"`
				Markets     []*state.Market `json:"markets"`
				Count       int             `json:"count"`
			}{
				EventTicker: eventTicker,
				Markets:     markets,
				Count:       len(markets),
			}
			totalMarkets += len(markets)
		}
		
		categories = append(categories, CategoryGroup{
			Category:     category,
			EventTickers: eventList,
			TotalMarkets: totalMarkets,
			Events:       eventDetails,
		})
	}
	
	response := struct {
		Categories []CategoryGroup `json:"categories"`
		Count      int             `json:"count"`
		Timestamp  time.Time       `json:"timestamp"`
	}{
		Categories: categories,
		Count:      len(categories),
		Timestamp:  time.Now(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func parseInt(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

