package alerting

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type DiscordClient struct {
	webhookURL string
	client     *http.Client
}

func NewDiscordClient(webhookURL string) *DiscordClient {
	return &DiscordClient{
		webhookURL: webhookURL,
		client:     &http.Client{},
	}
}

func (c *DiscordClient) Send(message string) error {
	payload := map[string]interface{}{
		"content": message,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", c.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

