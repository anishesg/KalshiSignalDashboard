package state

import "time"

type MarketStatus string

const (
	StatusInitialized MarketStatus = "initialized"
	StatusInactive    MarketStatus = "inactive"
	StatusActive      MarketStatus = "active"
	StatusClosed      MarketStatus = "closed"
	StatusDetermined  MarketStatus = "determined"
	StatusDisputed    MarketStatus = "disputed"
	StatusAmended     MarketStatus = "amended"
	StatusFinalized   MarketStatus = "finalized"
)

type Market struct {
	Ticker         string       `json:"ticker"`
	Title          string       `json:"title"`
	Category       string       `json:"category"`
	Status         MarketStatus `json:"status"`
	ExpirationTime *time.Time   `json:"expiration_time,omitempty"`
	EventTicker    string       `json:"event_ticker"`
	YesSubTitle    string       `json:"yes_sub_title,omitempty"`
	NoSubTitle     string       `json:"no_sub_title,omitempty"`
}

func (m *Market) Clone() *Market {
	var expTime *time.Time
	if m.ExpirationTime != nil {
		t := *m.ExpirationTime
		expTime = &t
	}
	return &Market{
		Ticker:         m.Ticker,
		Title:          m.Title,
		Category:       m.Category,
		Status:         m.Status,
		ExpirationTime: expTime,
		EventTicker:    m.EventTicker,
		YesSubTitle:    m.YesSubTitle,
		NoSubTitle:     m.NoSubTitle,
	}
}

