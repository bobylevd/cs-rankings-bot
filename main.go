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

	if strings.HasPrefix(m.Content, "!teams") {
		args := strings.Fields(m.Content)

		// Example: Pass guildID and voiceChannelID
		guildID := m.GuildID
		voiceChannelID := getVoiceChannelIDForUser(s, guildID, m.Author.ID) // Helper function to get the voice channel of the message author

		if voiceChannelID == "" {
			s.ChannelMessageSend(m.ChannelID, "You need to be in a voice channel!")
			return
		}

		// Check if the `-a` flag is set (take all players)
		takeAll := false
		if len(args) > 1 && args[1] == "-a" {
			takeAll = true
		}

		// Fetch and select players
		commentatorID := "108220450194092032"
		selectedPlayers, err := selectPlayersForGame(s, guildID, voiceChannelID, takeAll, commentatorID)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Error selecting players: %v", err))
			return
		}

		// Process and display teams (Team 1 and Team 2)
		team1, team2 := splitTeamsByELO(selectedPlayers)
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Team 1: %v\nTeam 2: %v", getTeamNames(team1), getTeamNames(team2)))
	}
}
