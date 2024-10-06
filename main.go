package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
	"os"
	"strings"
)

// a map to associate commands with their handler functions
var commandHandlers = map[string]func(s *discordgo.Session, m *discordgo.MessageCreate, args []string){
	"!teams": handleTeamsCommand,
	"!win":   handleWinCommand,
	"!end":   handleEndSessionCommand,
}

func main() {
	initDB()
	defer db.Close()

	token := os.Getenv("DISCORD_BOT_TOKEN")
	if token == "" {
		log.Fatal("No Discord bot token provided. Set the DISCORD_BOT_TOKEN environment variable.")
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal("Error creating Discord session,", err)
	}

	fmt.Println("Bot is running...")

	// Import historical data if needed (comment out if not needed in normal operations)
	//importHistoricalData("historical_data.json", dg)

	// Register the message handler
	dg.AddHandler(onMessageCreate)

	// Open the WebSocket and begin listening.
	err = dg.Open()

	if err != nil {
		log.Fatal("Error opening connection,", err)
	}
	defer dg.Close()

	// Keep the program running
	select {}
}

// Command handler function
func onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	args := strings.Fields(m.Content)
	if len(args) == 0 {
		return
	}

	if handler, exists := commandHandlers[args[0]]; exists {
		handler(s, m, args)
	}
}
