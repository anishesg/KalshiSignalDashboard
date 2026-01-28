package alerting

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kalshi-signal-feed/internal/config"
	"github.com/kalshi-signal-feed/internal/signals"
)

type Manager struct {
	config      config.AlertingConfig
	signalChan  <-chan signals.Signal
	slackClient *SlackClient
	discordClient *DiscordClient
	cooldown    map[string]time.Time
	mu          sync.RWMutex
}

func NewManager(cfg config.AlertingConfig, signalChan <-chan signals.Signal) *Manager {
	var slackClient *SlackClient
	var discordClient *DiscordClient

	if cfg.SlackWebhookURL != "" {
		slackClient = NewSlackClient(cfg.SlackWebhookURL)
	}

	if cfg.DiscordWebhookURL != "" {
		discordClient = NewDiscordClient(cfg.DiscordWebhookURL)
	}

	return &Manager{
		config:       cfg,
		signalChan:   signalChan,
		slackClient:  slackClient,
		discordClient: discordClient,
		cooldown:     make(map[string]time.Time),
	}
}

func (m *Manager) Run(ctx context.Context) error {
	if !m.config.Enabled {
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case signal := <-m.signalChan:
			if signal.Metadata.ThresholdCrossed {
				m.handleSignal(signal)
			}
		}
	}
}

func (m *Manager) handleSignal(signal signals.Signal) {
	// Check cooldown
	key := signal.MarketTicker + string(signal.Type)
	m.mu.RLock()
	lastAlert, inCooldown := m.cooldown[key]
	m.mu.RUnlock()

	if inCooldown {
		cooldownDuration := time.Duration(m.config.AlertCooldownSecs) * time.Second
		if time.Since(lastAlert) < cooldownDuration {
			return
		}
	}

	// Update cooldown
	m.mu.Lock()
	m.cooldown[key] = time.Now()
	m.mu.Unlock()

	// Send alerts
	message := m.formatSignalMessage(signal)

	if m.slackClient != nil {
		go m.slackClient.Send(message)
	}

	if m.discordClient != nil {
		go m.discordClient.Send(message)
	}
}

func (m *Manager) formatSignalMessage(signal signals.Signal) string {
	var msg string

	switch signal.Type {
	case signals.SignalTypeImpliedProbabilityDrift:
		if signal.ImpliedProbabilityDrift != nil {
			msg = fmt.Sprintf("🚨 **Implied Probability Drift**\n"+
				"Market: %s\n"+
				"Delta: %.2f%%\n"+
				"Drift: %.2fσ\n"+
				"Confidence: %.0f%%",
				signal.MarketTicker,
				signal.ImpliedProbabilityDrift.Delta*100,
				signal.Value,
				signal.Metadata.Confidence*100,
			)
		}

	case signals.SignalTypeOrderbookImbalance:
		if signal.OrderbookImbalance != nil {
			msg = fmt.Sprintf("⚖️ **Orderbook Imbalance**\n"+
				"Market: %s\n"+
				"Bid Ratio: %.2f\n"+
				"Spread: %d cents\n"+
				"Confidence: %.0f%%",
				signal.MarketTicker,
				signal.OrderbookImbalance.BidRatio,
				signal.OrderbookImbalance.SpreadCents,
				signal.Metadata.Confidence*100,
			)
		}

	case signals.SignalTypeVolumeSurge:
		if signal.VolumeSurge != nil {
			msg = fmt.Sprintf("📈 **Volume Surge**\n"+
				"Market: %s\n"+
				"Multiplier: %.2fx\n"+
				"Confidence: %.0f%%",
				signal.MarketTicker,
				signal.VolumeSurge.VolumeMultiplier,
				signal.Metadata.Confidence*100,
			)
		}
	}

	if msg == "" {
		msg = fmt.Sprintf("Signal: %s on %s (Value: %.2f)", signal.Type, signal.MarketTicker, signal.Value)
	}

	return msg
}

