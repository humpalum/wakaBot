package main

import (
	"fmt"
	"testing"
)

func TestSumoAPIIntegration(t *testing.T) {
	client := NewSumoClient()

	bashoID := CurrentBashoID()
	fmt.Println("Basho ID:", bashoID)

	resp, err := client.FetchTorikumi(bashoID, 1)
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
	wins, losses, err := client.FetchWakatakakageRecord(bashoID)
	if err != nil {
		t.Fatalf("FetchWakatakakageRecord error: %v", err)
	}
	fmt.Printf("Current record: %d-%d\n", wins, losses)
}
