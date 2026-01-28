package ingestion

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type GetSeriesResponse struct {
	Series []Series `json:"series"`
	Cursor *string  `json:"cursor"`
}

type Series struct {
	Ticker     string `json:"ticker"`
	Title      string `json:"title"`
	Category   string `json:"category"`
	Subtitle   string `json:"subtitle,omitempty"`
	SeriesType string `json:"series_type,omitempty"`
}

// fetchPoliticsSeries fetches all series in the Politics category
func (c *RESTClient) fetchPoliticsSeries(ctx context.Context) ([]string, error) {
	var allSeriesTickers []string
	cursor := (*string)(nil)

	for {
		url := c.baseURL + "/series"
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, err
		}

		q := req.URL.Query()
		q.Set("category", "Politics")
		q.Set("limit", "100")
		if cursor != nil {
			q.Set("cursor", *cursor)
		}
		req.URL.RawQuery = q.Encode()

		resp, err := c.client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("failed to fetch series: status %d, body: %s", resp.StatusCode, string(body))
		}

		var seriesResp GetSeriesResponse
		if err := json.NewDecoder(resp.Body).Decode(&seriesResp); err != nil {
			return nil, err
		}

		// Collect series tickers
		for _, s := range seriesResp.Series {
			allSeriesTickers = append(allSeriesTickers, s.Ticker)
		}

		// Check if there are more pages
		if seriesResp.Cursor == nil || *seriesResp.Cursor == "" {
			break
		}
		cursor = seriesResp.Cursor
	}

	return allSeriesTickers, nil
}

