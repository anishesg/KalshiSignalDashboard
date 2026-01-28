package alerting

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type SlackClient struct {
	webhookURL string
	client     *http.Client
}

func NewSlackClient(webhookURL string) *SlackClient {
	return &SlackClient{
		webhookURL: webhookURL,
		client:     &http.Client{},
	}
}

func (c *SlackClient) Send(message string) error {
	payload := map[string]string{
		"text": message,
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

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

