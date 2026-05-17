package main

import (
	"encoding/json"
	"fmt"
	"net/http"
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
	ID            string `json:"id"`
	BashoID       string `json:"bashoId"`
	Division      string `json:"division"`
	Day           int    `json:"day"`
	MatchNo       int    `json:"matchNo"`
	EastID        int    `json:"eastId"`
	EastShikona   string `json:"eastShikona"`
	EastRank      string `json:"eastRank"`
	WestID        int    `json:"westId"`
	WestShikona   string `json:"westShikona"`
	WestRank      string `json:"westRank"`
	Kimarite      string `json:"kimarite"`
	WinnerID      int    `json:"winnerId"`
	WinnerEn      string `json:"winnerEn"`
	WinnerJp      string `json:"winnerJp"`
}

type BanzukeEntry struct {
	Side       string `json:"side"`
	RikishiID  int    `json:"rikishiID"`
	ShikonaEn  string `json:"shikonaEn"`
	Rank       string `json:"rank"`
	Wins       int    `json:"wins"`
	Losses     int    `json:"losses"`
	Absences   int    `json:"absences"`
}

type SumoClient struct {
	http *http.Client
}

func NewSumoClient() *SumoClient {
	return &SumoClient{
		http: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *SumoClient) FetchTorikumi(bashoID string, day int) (*TorikumiResponse, error) {
	url := fmt.Sprintf("%s/basho/%s/torikumi/Makuuchi/%d", baseURL, bashoID, day)
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

type BanzukeResponse struct {
	BashoID  string         `json:"bashoId"`
	Division string         `json:"division"`
	East     []BanzukeEntry `json:"east"`
	West     []BanzukeEntry `json:"west"`
}

func (c *SumoClient) FetchBanzuke(bashoID string) ([]BanzukeEntry, error) {
	url := fmt.Sprintf("%s/basho/%s/banzuke/Makuuchi", baseURL, bashoID)
	resp, err := c.http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching banzuke: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("banzuke API returned status %d", resp.StatusCode)
	}

	var result BanzukeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding banzuke: %w", err)
	}
	return append(result.East, result.West...), nil
}

func (c *SumoClient) FetchWakatakakageRecord(bashoID string) (wins, losses int, err error) {
	entries, err := c.FetchBanzuke(bashoID)
	if err != nil {
		return 0, 0, err
	}
	for _, e := range entries {
		if e.RikishiID == wakatakakageID {
			return e.Wins, e.Losses, nil
		}
	}
	return 0, 0, fmt.Errorf("Wakatakakage not found in banzuke")
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
func TournamentDay(startDate time.Time) int {
	now := time.Now().UTC().Truncate(24 * time.Hour)
	start := startDate.UTC().Truncate(24 * time.Hour)
	days := int(now.Sub(start).Hours()/24) + 1
	if days < 1 {
		return 0
	}
	if days > 15 {
		return 15
	}
	return days
}
