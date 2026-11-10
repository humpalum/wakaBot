package main

import (
	"fmt"
	"testing"
	"time"
)

func TestSumoAPIIntegration(t *testing.T) {
	client := NewSumoClient()

	bashoID := CurrentBashoID()
	fmt.Println("Basho ID:", bashoID)

	division, err := client.FetchDivision()
	if err != nil {
		t.Fatalf("FetchDivision error: %v", err)
	}
	fmt.Println("Division:", division)

	resp, err := client.FetchTorikumi(bashoID, division, 1)
	if err != nil {
		t.Fatalf("FetchTorikumi error: %v", err)
	}
	fmt.Println("Start date:", resp.StartDate)
	fmt.Println("Tournament day:", TournamentDay(resp.StartDate))
	fmt.Printf("Day 1 bouts: %d\n", len(resp.Torikumi))

	if len(resp.Torikumi) == 0 {
		t.Fatal("No bouts returned")
	}

	// Find Wakatakakage in day 1
	found := false
	for _, b := range resp.Torikumi {
		if b.EastID == wakatakakageID || b.WestID == wakatakakageID {
			found = true
			fmt.Printf("Day 1 match: #%d | %s (%s) vs %s (%s) | Winner: %s | Kimarite: %s\n",
				b.MatchNo, b.EastShikona, b.EastRank, b.WestShikona, b.WestRank, b.WinnerEn, b.Kimarite)
		}
	}
	if !found {
		t.Log("Wakatakakage not found in day 1 (may be absent)")
	}

	// Fetch record
	wins, losses, err := client.FetchWakatakakageRecord(bashoID, division)
	if err != nil {
		t.Fatalf("FetchWakatakakageRecord error: %v", err)
	}
	fmt.Printf("Current record: %d-%d\n", wins, losses)
}

func TestRankDivision(t *testing.T) {
	for rank, want := range map[string]string{
		"Juryo 5 West":      "Juryo",
		"Maegashira 3 East": "Makuuchi",
		"Ozeki 1 West":      "Makuuchi",
		"Makushita 2 East":  "Makushita",
	} {
		if got := rankDivision(rank); got != want {
			t.Errorf("rankDivision(%q) = %q, want %q", rank, got, want)
		}
	}
}

func TestTournamentDay(t *testing.T) {
	now := time.Now().UTC()
	for name, c := range map[string]struct {
		start time.Time
		want  int
	}{
		"day 1":       {now, 1},
		"day 15":      {now.AddDate(0, 0, -14), 15},
		"ended":       {now.AddDate(0, 0, -15), 0},
		"not started": {now.AddDate(0, 0, 1), 0},
		"no date yet": {time.Time{}, 0},
	} {
		if got := TournamentDay(c.start); got != c.want {
			t.Errorf("%s: TournamentDay = %d, want %d", name, got, c.want)
		}
	}
}
