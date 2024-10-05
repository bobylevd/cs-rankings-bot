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
