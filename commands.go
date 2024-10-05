package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"strings"
)

// Command to select teams and balance by ELO
func selectTeamsCommand(s *discordgo.Session, channelID string) {
	selectedPlayers, err := selectPlayersForGame()
	if err != nil {
		s.ChannelMessageSend(channelID, "Error selecting players.")
		return
	}
	team1, team2 := splitTeamsByELO(selectedPlayers)
	s.ChannelMessageSend(channelID, fmt.Sprintf("Team 1: %v\nTeam 2: %v", getTeamNames(team1), getTeamNames(team2)))
}

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
