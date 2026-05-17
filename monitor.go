package main

import (
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
)

type BoutState int

const (
	StateIdle BoutState = iota
	StateUpcomingSent
	StateResultSent
)

type Monitor struct {
	sumo      *SumoClient
	discord   *discordgo.Session
	channelID string
	roleID    string

	state    BoutState
	stateDay int // track which tournament day the state belongs to
}

func NewMonitor(sumo *SumoClient, discord *discordgo.Session, channelID, roleID string) *Monitor {
	return &Monitor{
		sumo:      sumo,
		discord:   discord,
		channelID: channelID,
		roleID:    roleID,
		state:     StateIdle,
	}
}

func (m *Monitor) Run(stop <-chan struct{}) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Run immediately on start, then on each tick
	m.poll()
	for {
		select {
		case <-ticker.C:
			m.poll()
		case <-stop:
			log.Println("Monitor stopped")
			return
		}
	}
}

func (m *Monitor) poll() {
	bashoID := CurrentBashoID()

	// First fetch day 1 to get the start date and determine current day
	resp, err := m.sumo.FetchTorikumi(bashoID, 1)
	if err != nil {
		log.Printf("Error fetching day 1 torikumi: %v", err)
		return
	}

	day := TournamentDay(resp.StartDate)
	if day < 1 {
		log.Println("Tournament hasn't started yet")
		return
	}

	// Reset state when the day changes
	if day != m.stateDay {
		m.state = StateIdle
		m.stateDay = day
		log.Printf("New tournament day: %d", day)
	}

	if m.state == StateResultSent {
		return // nothing more to do today
	}

	// Fetch today's torikumi
	todayResp, err := m.sumo.FetchTorikumi(bashoID, day)
	if err != nil {
		log.Printf("Error fetching day %d torikumi: %v", day, err)
		return
	}

	// Find Wakatakakage's bout
	var wakataBout *Bout
	var prevBout *Bout
	for i := range todayResp.Torikumi {
		b := &todayResp.Torikumi[i]
		if b.EastID == wakatakakageID || b.WestID == wakatakakageID {
			wakataBout = b
			// Find the previous bout by matchNo
			for j := range todayResp.Torikumi {
				p := &todayResp.Torikumi[j]
				if p.MatchNo == b.MatchNo-1 {
					prevBout = p
					break
				}
			}
			break
		}
	}

	if wakataBout == nil {
		log.Printf("Wakatakakage not found in day %d torikumi (may be absent)", day)
		return
	}

	boutCompleted := wakataBout.WinnerID != 0
	prevCompleted := prevBout == nil || prevBout.WinnerID != 0 // nil means he's first match

	// Determine opponent info
	opponentName := wakataBout.WestShikona
	opponentRank := wakataBout.WestRank
	if wakataBout.WestID == wakatakakageID {
		opponentName = wakataBout.EastShikona
		opponentRank = wakataBout.EastRank
	}

	log.Printf("Day %d | Match #%d | vs %s | prev_done=%v | bout_done=%v | state=%d",
		day, wakataBout.MatchNo, opponentName, prevCompleted, boutCompleted, m.state)

	// Check if bout is upcoming (previous bout done, his bout not yet)
	if !boutCompleted && prevCompleted && m.state == StateIdle {
		m.sendUpcoming(opponentName, opponentRank)
		m.state = StateUpcomingSent
		return
	}

	// Check if bout is completed
	if boutCompleted && m.state != StateResultSent {
		m.sendResult(bashoID, wakataBout, opponentName)
		m.state = StateResultSent
		return
	}
}

func (m *Monitor) sendUpcoming(opponent, opponentRank string) {
	msg := fmt.Sprintf("<@&%s> **Wakatakakage's bout is up next!** He faces **%s** (%s).",
		m.roleID, opponent, opponentRank)

	_, err := m.discord.ChannelMessageSend(m.channelID, msg)
	if err != nil {
		log.Printf("Error sending upcoming message: %v", err)
		return
	}
	log.Println("Sent upcoming bout notification")
}

func (m *Monitor) sendResult(bashoID string, bout *Bout, opponent string) {
	won := bout.WinnerID == wakatakakageID

	wins, losses, err := m.sumo.FetchWakatakakageRecord(bashoID)
	if err != nil {
		log.Printf("Error fetching record: %v", err)
		// Fall back to sending without record
		wins, losses = -1, -1
	}

	var msg string
	if won {
		msg = fmt.Sprintf("**Wakatakakage wins!** He defeated **%s** by **%s**.",
			opponent, bout.Kimarite)
	} else {
		msg = fmt.Sprintf("**Wakatakakage loses.** He was defeated by **%s** via **%s**.",
			opponent, bout.Kimarite)
	}

	if wins >= 0 {
		msg += fmt.Sprintf(" Current record: **%d-%d**.", wins, losses)
	}

	_, err = m.discord.ChannelMessageSend(m.channelID, msg)
	if err != nil {
		log.Printf("Error sending result message: %v", err)
		return
	}
	log.Println("Sent bout result notification")
}
