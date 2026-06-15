// Package odds fetches market-consensus h2h betting odds for WC2026 matches
// from The Odds API (the-odds-api.com). The free tier allows 500 requests/month;
// one daily sync uses ~31 requests for the tournament month.
package odds

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

const oddsAPIURL = "https://api.the-odds-api.com/v4/sports/soccer_fifa_world_cup/odds"

// Client calls The Odds API.
type Client struct {
	key  string
	http *http.Client
}

// New returns a Client for the given API key.
func New(key string) *Client {
	return &Client{key: key, http: &http.Client{Timeout: 20 * time.Second}}
}

// Bookmaker is one bookmaker's odds for an event.
type Bookmaker struct {
	Key     string   `json:"key"`
	Markets []Market `json:"markets"`
}

// Market is a betting market (we only use "h2h").
type Market struct {
	Key      string    `json:"key"`
	Outcomes []Outcome `json:"outcomes"`
}

// Outcome is one possible result with a decimal price.
type Outcome struct {
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

// OddsEvent is a single match from The Odds API.
type OddsEvent struct {
	ID           string      `json:"id"`
	HomeTeam     string      `json:"home_team"`
	AwayTeam     string      `json:"away_team"`
	CommenceTime time.Time   `json:"commence_time"`
	Bookmakers   []Bookmaker `json:"bookmakers"`
}

// FetchOdds returns all upcoming WC2026 h2h odds from The Odds API.
func (c *Client) FetchOdds(ctx context.Context) ([]OddsEvent, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, oddsAPIURL, nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	q.Set("apiKey", c.key)
	q.Set("regions", "eu")
	q.Set("markets", "h2h")
	q.Set("oddsFormat", "decimal")
	req.URL.RawQuery = q.Encode()

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("odds api request: %w", err)
	}
	defer resp.Body.Close()

	if remaining := resp.Header.Get("x-requests-remaining"); remaining != "" {
		log.Printf("[odds] requests remaining: %s", remaining)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("odds api status %d", resp.StatusCode)
	}

	var events []OddsEvent
	if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
		return nil, fmt.Errorf("odds api decode: %w", err)
	}
	return events, nil
}
