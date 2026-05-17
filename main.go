package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

func main() {
	token := os.Getenv("DISCORD_TOKEN")
	channelID := os.Getenv("DISCORD_CHANNEL_ID")
	roleID := os.Getenv("DISCORD_ROLE_ID")

	if token == "" || channelID == "" || roleID == "" {
		log.Fatal("Required environment variables: DISCORD_TOKEN, DISCORD_CHANNEL_ID, DISCORD_ROLE_ID")
	}

	// Create Discord session
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatalf("Error creating Discord session: %v", err)
	}

	dg.Identify.Intents = discordgo.IntentsGuildMessages

	if err := dg.Open(); err != nil {
		log.Fatalf("Error opening Discord connection: %v", err)
	}
	defer dg.Close()

	log.Printf("Bot connected. Monitoring Wakatakakage's bouts in channel %s", channelID)
	log.Printf("Current basho: %s", CurrentBashoID())

	sumo := NewSumoClient()
	monitor := NewMonitor(sumo, dg, channelID, roleID)

	stop := make(chan struct{})
	go monitor.Run(stop)

	// Wait for shutdown signal
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
	<-sc

	log.Println("Shutting down...")
	close(stop)
}
