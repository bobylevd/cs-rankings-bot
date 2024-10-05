package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
	"os"
	"strings"
)

func main() {
	initDB() // Initialize database
	defer db.Close()

	// Initialize Discord bot
	token := os.Getenv("DISCORD_BOT_TOKEN")
	if token == "" {
		log.Fatal("No Discord bot token provided. Set the DISCORD_BOT_TOKEN environment variable.")
	}
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal("Error creating Discord session,", err)
	}

	// Import historical data if needed (comment out if not needed in normal operations)
	// importHistoricalData("historical_data.json", dg)

	fmt.Println("Bot is running...")
	dg.AddHandler(onMessageCreate)
	//importHistoricalData("historical_data.json", dg)
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

	content := strings.TrimSpace(m.Content)

	// Handle commands
	if strings.HasPrefix(content, "!selectteams") {
		selectTeamsCommand(s, m.ChannelID)
	} else if strings.HasPrefix(content, "!playerStats") {
		args := strings.Fields(content)
		if len(args) < 2 {
			s.ChannelMessageSend(m.ChannelID, "Usage: !playerStats <playerID>")
			return
		}
		playerStatsCommand(s, m.ChannelID, args[1])
	} else if strings.HasPrefix(content, "!eloGraph") {
		args := strings.Fields(content)
		if len(args) < 2 {
			s.ChannelMessageSend(m.ChannelID, "Usage: !eloGraph <playerID>")
			return
		}
		eloGraphCommand(s, m.ChannelID, args[1])
	}
}
