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
	var day1 *Bout
	for i, b := range resp.Torikumi {
		if b.EastID == wakatakakageID || b.WestID == wakatakakageID {
			day1 = &resp.Torikumi[i]
			fmt.Printf("Day 1 match: #%d | %s (%s) vs %s (%s) | Winner: %s | Kimarite: %s\n",
				b.MatchNo, b.EastShikona, b.EastRank, b.WestShikona, b.WestRank, b.WinnerEn, b.Kimarite)
		}
	}
	if day1 == nil {
		t.Skip("Wakatakakage not found in day 1 (may be absent)")
	}

	wins, losses, err := client.FetchWakatakakageRecord(bashoID, *day1)
	if err != nil {
		t.Fatalf("FetchWakatakakageRecord error: %v", err)
	}
	fmt.Printf("Record after day 1: %d-%d\n", wins, losses)
}

func TestTally(t *testing.T) {
	past := []Bout{
		{Day: 1, WinnerID: wakatakakageID},
		{Day: 2, WinnerID: 99},
		{Day: 3, WinnerID: 0},  // stale: aggregated data not caught up with today's result
		{Day: 4, WinnerID: 99}, // later day must not count
	}
	if w, l := tally(past, Bout{Day: 3, WinnerID: wakatakakageID}); w != 2 || l != 1 {
		t.Errorf("tally = %d-%d, want 2-1", w, l)
	}
	past[2].WinnerID = wakatakakageID // already caught up: no double count
	if w, l := tally(past, Bout{Day: 3, WinnerID: wakatakakageID}); w != 2 || l != 1 {
		t.Errorf("tally = %d-%d, want 2-1", w, l)
	}
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
