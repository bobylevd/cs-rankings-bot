package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
	"os"
	"strings"
)

func main() {
	// Initialize the database
	db, err := InitDB("match_data.db")
	if err != nil {
		log.Fatalf("Error initializing database: %v", err)
	}
	defer db.Close()

	// Get the Discord bot token from the environment variable
	token := os.Getenv("DISCORD_BOT_TOKEN")
	if token == "" {
		log.Fatal("No Discord bot token provided. Set the DISCORD_BOT_TOKEN environment variable.")
	}

	// Create a new Discord session using the provided bot token
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatalf("Error creating Discord session: %v", err)
	}
	defer dg.Close()

	// Open the Discord session
	err = dg.Open()
	if err != nil {
		log.Fatalf("Error opening Discord session: %v", err)
	}

	fmt.Println("Bot is running...")

	// Create a Discord instance from the session
	discordInstance := NewDiscord(dg)

	hasMatches, err := db.HasMatches()
	if err != nil {
		log.Fatalf("Error checking for existing matches: %v", err)
	}

	if !hasMatches {
		// Import historical data if no matches exist
		fmt.Println("No existing matches found. Importing historical data...")
		// Replace "historical_data.json" with the path to your historical data file
		ImportHistoricalData("historical_data.json", db, discordInstance)
		fmt.Println("Historical data import completed.")
	} else {
		fmt.Println("Existing matches found. Skipping historical data import.")
	}

	// Register the message handler
	dg.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		onMessageCreate(s, m, db, discordInstance)
	})

	// Register session handler
	dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		handleInteraction(s, i, db)
	})

	// Keep the program running
	select {}
}

// Command handler function
func onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate, db *DB, discordInstance *Discord) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	args := strings.Fields(m.Content)
	if len(args) == 0 {
		return
	}

	command := strings.ToLower(args[0])

	switch command {
	case "!teams":
		handleTeamsCommand(s, m, args, db, discordInstance)
	case "!win":
		handleWinCommand(s, m, args, db)
	//case "!end":
	//	handleEndSessionCommand(s, m, args, db)
	case "!stats":
		playerStatsCommand(s, m, args, db, discordInstance)
	case "!elograph":
		playerID := m.Author.ID
		eloGraphCommand(s, m.ChannelID, playerID, db)
	}
}
