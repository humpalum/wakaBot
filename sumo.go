package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	baseURL        = "https://www.sumo-api.com/api"
	wakatakakageID = 12
)

type TorikumiResponse struct {
	Date      string    `json:"date"`
	StartDate time.Time `json:"startDate"`
	EndDate   time.Time `json:"endDate"`
	Torikumi  []Bout    `json:"torikumi"`
}

type Bout struct {
	ID          string `json:"id"`
	BashoID     string `json:"bashoId"`
	Division    string `json:"division"`
	Day         int    `json:"day"`
	MatchNo     int    `json:"matchNo"`
	EastID      int    `json:"eastId"`
	EastShikona string `json:"eastShikona"`
	EastRank    string `json:"eastRank"`
	WestID      int    `json:"westId"`
	WestShikona string `json:"westShikona"`
	WestRank    string `json:"westRank"`
	Kimarite    string `json:"kimarite"`
	WinnerID    int    `json:"winnerId"`
	WinnerEn    string `json:"winnerEn"`
	WinnerJp    string `json:"winnerJp"`
}

type SumoClient struct {
	http *http.Client
}

func NewSumoClient() *SumoClient {
	return &SumoClient{
		http: &http.Client{Timeout: 10 * time.Second},
	}
}

// FetchDivision returns Wakatakakage's current division as used by the API paths.
func (c *SumoClient) FetchDivision() (string, error) {
	resp, err := c.http.Get(fmt.Sprintf("%s/rikishi/%d", baseURL, wakatakakageID))
	if err != nil {
		return "", fmt.Errorf("fetching rikishi: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("rikishi API returned status %d", resp.StatusCode)
	}

	var r struct {
		CurrentRank string `json:"currentRank"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return "", fmt.Errorf("decoding rikishi: %w", err)
	}
	return rankDivision(r.CurrentRank), nil
}

// rankDivision maps a rank like "Juryo 5 West" or "Maegashira 3 East" to its division.
func rankDivision(rank string) string {
	switch d, _, _ := strings.Cut(rank, " "); d {
	case "Juryo", "Makushita", "Sandanme", "Jonidan", "Jonokuchi":
		return d
	default: // Yokozuna, Ozeki, Sekiwake, Komusubi, Maegashira
		return "Makuuchi"
	}
}

func (c *SumoClient) FetchTorikumi(bashoID, division string, day int) (*TorikumiResponse, error) {
	url := fmt.Sprintf("%s/basho/%s/torikumi/%s/%d", baseURL, bashoID, division, day)
	resp, err := c.http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching torikumi: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("torikumi API returned status %d", resp.StatusCode)
	}

	var result TorikumiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding torikumi: %w", err)
	}
	return &result, nil
}

// FetchWakatakakageRecord returns his basho record including today's bout.
// Today's result comes from the torikumi we already hold: aggregated data (banzuke
// wins/losses) lags behind it, which made the bot repeat the previous day's record.
func (c *SumoClient) FetchWakatakakageRecord(bashoID string, today Bout) (wins, losses int, err error) {
	resp, err := c.http.Get(fmt.Sprintf("%s/rikishi/%d/matches?bashoId=%s", baseURL, wakatakakageID, bashoID))
	if err != nil {
		return 0, 0, fmt.Errorf("fetching matches: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("matches API returned status %d", resp.StatusCode)
	}

	var result struct {
		Records []Bout `json:"records"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, 0, fmt.Errorf("decoding matches: %w", err)
	}
	wins, losses = tally(result.Records, today)
	return wins, losses, nil
}

// tally counts results of bouts before today's day, plus today's bout.
func tally(past []Bout, today Bout) (wins, losses int) {
	bouts := []Bout{today}
	for _, b := range past {
		if b.Day < today.Day {
			bouts = append(bouts, b)
		}
	}
	for _, b := range bouts {
		switch b.WinnerID {
		case 0: // not fought yet
		case wakatakakageID:
			wins++
		default:
			losses++
		}
	}
	return wins, losses
}

// CurrentBashoID returns the basho ID for the current tournament period.
// Basho months: January(01), March(03), May(05), July(07), September(09), November(11).
func CurrentBashoID() string {
	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	// Find the most recent or current basho month
	bashoMonths := []int{1, 3, 5, 7, 9, 11}
	bashoMonth := bashoMonths[0]
	for _, m := range bashoMonths {
		if month >= m {
			bashoMonth = m
		}
	}
	return fmt.Sprintf("%d%02d", year, bashoMonth)
}

// TournamentDay computes which day of the tournament it is (1-15) from the start date.
// Returns 0 outside the tournament, including when the API has no start date yet (zero time).
func TournamentDay(startDate time.Time) int {
	now := time.Now().UTC().Truncate(24 * time.Hour)
	start := startDate.UTC().Truncate(24 * time.Hour)
	days := int(now.Sub(start).Hours()/24) + 1
	if days < 1 || days > 15 {
		return 0
	}
	return days
}
