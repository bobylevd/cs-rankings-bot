package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
	"strings"
)

// Command to display player stats
func playerStatsCommand(s *discordgo.Session, channelID, playerID string) {
	player, err := getPlayerStats(playerID)
	if err != nil {
		s.ChannelMessageSend(channelID, fmt.Sprintf("Error fetching player stats: %v", err))
		return
	}
	stats := fmt.Sprintf(
		"Player: %s\nGames Played: %d\nWins: %d\nELO: %d\nKills: %d\nAssists: %d\nDeaths: %d",
		player.PlayerName, player.GamesPlayed, player.Wins, player.ELO, player.Kills, player.Assists, player.Deaths,
	)
	s.ChannelMessageSend(channelID, stats)
}

// Command to display ELO graph data (for graphing or text output)
func eloGraphCommand(s *discordgo.Session, channelID, playerID string) {
	elos, timestamps, err := getPlayerELOHistory(playerID)
	if err != nil {
		s.ChannelMessageSend(channelID, fmt.Sprintf("Error fetching ELO history: %v", err))
		return
	}

	message := fmt.Sprintf("ELO History for player %s:\n", playerID)
	for i := range elos {
		message += fmt.Sprintf("%s: %d ELO\n", timestamps[i], elos[i])
	}
	s.ChannelMessageSend(channelID, message)
}

func getTeamNames(players []*Player) string {
	var names []string
	for _, player := range players {
		names = append(names, player.PlayerName)
	}
	return strings.Join(names, ", ")
}

// Helper function to get the voice channel ID for a user
func getVoiceChannelIDForUser(s *discordgo.Session, guildID string, userID string) string {
	guild, err := s.State.Guild(guildID)
	if err != nil {
		log.Printf("Error getting guild: %v", err)
		return ""
	}

	// Loop through voice states to find the user’s voice channel
	for _, vs := range guild.VoiceStates {
		if vs.UserID == userID {
			return vs.ChannelID // Return the user's voice channel ID
		}
	}
	return ""
}

func handleTeamsCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	guildID := m.GuildID
	voiceChannelID := getVoiceChannelIDForUser(s, guildID, m.Author.ID)

	if voiceChannelID == "" {
		s.ChannelMessageSend(m.ChannelID, "You need to be in a voice channel!")
		return
	}

	// Check if the `-a` flag is set (take all players)
	takeAll := false
	if len(args) > 1 && args[1] == "-a" {
		takeAll = true
	}

	commentatorID := "108220450194092032"

	// Use helper function to get players
	playerIDs, err := GetPlayerIDsFromVoiceChannel(s, guildID, voiceChannelID)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Error: %v", err))
		return
	}

	team1, team2, err := selectPlayersForGame(playerIDs, takeAll, commentatorID)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Error selecting players: %v", err))
		return
	}

	// Store teams in DB and display them
	storeTeamsInDB(team1, team2)
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Team 1: %v\nTeam 2: %v", getTeamNames(team1), getTeamNames(team2)))
}

func handleWinCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) < 2 {
		s.ChannelMessageSend(m.ChannelID, "Please specify the winning team (team1won or team2won).")
		return
	}

	var winningTeam int
	if args[1] == "team1" {
		winningTeam = 1
	} else if args[1] == "team2" {
		winningTeam = 2
	} else {
		s.ChannelMessageSend(m.ChannelID, "Invalid team. Use team1won or team2won.")
		return
	}

	// Retrieve stored teams from the database
	team1, team2, err := getStoredTeamsFromDB()
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Error: %v. Please run `!teams` to form new teams.", err))
		return
	}

	// Save the match result using the stored teams
	matchID := saveMatchData(team1, team2, winningTeam)
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Match %d reported: Team %d won!", matchID, winningTeam))
}

func handleEndSessionCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	clearStoredTeamsFromDB()
	s.ChannelMessageSend(m.ChannelID, "The current match session has ended, and teams have been cleared.")
}
