package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Kalshi    KalshiConfig
	Ingestion IngestionConfig
	Signals   SignalConfig
	API       APIConfig
	Alerting  AlertingConfig
}

type KalshiConfig struct {
	APIBaseURL      string
	WebSocketURL    string
	APIKeyID        string
	PrivateKeyPath  string
}

type IngestionConfig struct {
	WebSocketReconnectDelaySecs int
	RESTPollIntervalSecs        int
	RateLimitPerSecond           int
}

type SignalConfig struct {
	ComputationIntervalSecs int
	DriftWindowSecs         int
	DriftThreshold           float64
	ImbalanceThreshold      float64
	VolumeSurgeThreshold    float64
	VolumeWindowSecs         int
}

type APIConfig struct {
	BindAddress string
	CORSOrigins []string
}

type AlertingConfig struct {
	Enabled            bool
	SlackWebhookURL    string
	DiscordWebhookURL  string
	AlertCooldownSecs  int
}

func Load() (*Config, error) {
	cfg := &Config{
		Kalshi: KalshiConfig{
			APIBaseURL:     getEnv("KALSHI__KALSHI__API_BASE_URL", "https://api.elections.kalshi.com/trade-api/v2"),
			WebSocketURL:   getEnv("KALSHI__KALSHI__WEBSOCKET_URL", "wss://api.elections.kalshi.com/trade-api/v2/ws"),
			APIKeyID:       getEnv("KALSHI__KALSHI__API_KEY_ID", ""),
			PrivateKeyPath: getEnv("KALSHI__KALSHI__PRIVATE_KEY_PATH", "market_signal_bot.txt"),
		},
		Ingestion: IngestionConfig{
			WebSocketReconnectDelaySecs: getEnvInt("KALSHI__INGESTION__WEBSOCKET_RECONNECT_DELAY_SECS", 5),
			RESTPollIntervalSecs:        getEnvInt("KALSHI__INGESTION__REST_POLL_INTERVAL_SECS", 60),
			RateLimitPerSecond:          getEnvInt("KALSHI__INGESTION__RATE_LIMIT_PER_SECOND", 10),
		},
		Signals: SignalConfig{
			ComputationIntervalSecs: getEnvInt("KALSHI__SIGNALS__COMPUTATION_INTERVAL_SECS", 1),
			DriftWindowSecs:         getEnvInt("KALSHI__SIGNALS__DRIFT_WINDOW_SECS", 60),
			DriftThreshold:          getEnvFloat("KALSHI__SIGNALS__DRIFT_THRESHOLD", 2.0),
			ImbalanceThreshold:      getEnvFloat("KALSHI__SIGNALS__IMBALANCE_THRESHOLD", 0.3),
			VolumeSurgeThreshold:    getEnvFloat("KALSHI__SIGNALS__VOLUME_SURGE_THRESHOLD", 3.0),
			VolumeWindowSecs:         getEnvInt("KALSHI__SIGNALS__VOLUME_WINDOW_SECS", 30),
		},
		API: APIConfig{
			BindAddress: getEnv("KALSHI__API__BIND_ADDRESS", "0.0.0.0:8080"),
			CORSOrigins: getEnvSlice("KALSHI__API__CORS_ORIGINS", []string{"http://localhost:3000"}),
		},
		Alerting: AlertingConfig{
			Enabled:           getEnvBool("KALSHI__ALERTING__ENABLED", true),
			SlackWebhookURL:   getEnv("KALSHI__ALERTING__SLACK_WEBHOOK_URL", ""),
			DiscordWebhookURL: getEnv("KALSHI__ALERTING__DISCORD_WEBHOOK_URL", ""),
			AlertCooldownSecs: getEnvInt("KALSHI__ALERTING__ALERT_COOLDOWN_SECS", 300),
		},
	}

	// Load TOML config file if it exists
	tomlPath := "config/default.toml"
	if _, err := os.Stat(tomlPath); err == nil {
		data, err := os.ReadFile(tomlPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}

		var tomlConfig struct {
			Kalshi    map[string]interface{} `toml:"kalshi"`
			Ingestion map[string]interface{} `toml:"ingestion"`
			Signals   map[string]interface{} `toml:"signals"`
			API       map[string]interface{} `toml:"api"`
			Alerting  map[string]interface{} `toml:"alerting"`
		}

		if err := toml.Unmarshal(data, &tomlConfig); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}

		// Override with TOML values
		if kalshi, ok := tomlConfig.Kalshi["api_base_url"].(string); ok {
			cfg.Kalshi.APIBaseURL = kalshi
		}
		if kalshi, ok := tomlConfig.Kalshi["websocket_url"].(string); ok {
			cfg.Kalshi.WebSocketURL = kalshi
		}
		if kalshi, ok := tomlConfig.Ingestion["websocket_reconnect_delay_secs"].(int64); ok {
			cfg.Ingestion.WebSocketReconnectDelaySecs = int(kalshi)
		}
		if kalshi, ok := tomlConfig.Ingestion["rest_poll_interval_secs"].(int64); ok {
			cfg.Ingestion.RESTPollIntervalSecs = int(kalshi)
		}
		if kalshi, ok := tomlConfig.Ingestion["rate_limit_per_second"].(int64); ok {
			cfg.Ingestion.RateLimitPerSecond = int(kalshi)
		}
		if sig, ok := tomlConfig.Signals["computation_interval_secs"].(int64); ok {
			cfg.Signals.ComputationIntervalSecs = int(sig)
		}
		if sig, ok := tomlConfig.Signals["drift_window_secs"].(int64); ok {
			cfg.Signals.DriftWindowSecs = int(sig)
		}
		if sig, ok := tomlConfig.Signals["drift_threshold"].(float64); ok {
			cfg.Signals.DriftThreshold = sig
		}
		if sig, ok := tomlConfig.Signals["imbalance_threshold"].(float64); ok {
			cfg.Signals.ImbalanceThreshold = sig
		}
		if sig, ok := tomlConfig.Signals["volume_surge_threshold"].(float64); ok {
			cfg.Signals.VolumeSurgeThreshold = sig
		}
		if sig, ok := tomlConfig.Signals["volume_window_secs"].(int64); ok {
			cfg.Signals.VolumeWindowSecs = int(sig)
		}
		if api, ok := tomlConfig.API["bind_address"].(string); ok {
			cfg.API.BindAddress = api
		}
		if api, ok := tomlConfig.API["cors_origins"].([]interface{}); ok {
			origins := make([]string, 0, len(api))
			for _, v := range api {
				if s, ok := v.(string); ok {
					origins = append(origins, s)
				}
			}
			cfg.API.CORSOrigins = origins
		}
		if alert, ok := tomlConfig.Alerting["enabled"].(bool); ok {
			cfg.Alerting.Enabled = alert
		}
		if alert, ok := tomlConfig.Alerting["alert_cooldown_secs"].(int64); ok {
			cfg.Alerting.AlertCooldownSecs = int(alert)
		}
	}

	// Validate private key path
	if cfg.Kalshi.PrivateKeyPath != "" {
		if _, err := os.Stat(cfg.Kalshi.PrivateKeyPath); err != nil {
			// Try relative to current directory
			if _, err := os.Stat(filepath.Join(".", cfg.Kalshi.PrivateKeyPath)); err == nil {
				cfg.Kalshi.PrivateKeyPath = filepath.Join(".", cfg.Kalshi.PrivateKeyPath)
			}
		}
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}

