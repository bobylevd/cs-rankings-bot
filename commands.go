package main

import (
	"database/sql"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
	"time"
)

// Command to display player stats
func playerStatsCommand(s *discordgo.Session, channelID, playerID string, db *DB) {
	player, err := db.GetPlayer(playerID)
	if err != nil {
		s.ChannelMessageSend(channelID, fmt.Sprintf("Error fetching player stats: %v", err))
		return
	}
	stats := fmt.Sprintf(
		"Player: %s\nGames Played: %d\nWins: %d\nMMR: %d\nKills: %d\nAssists: %d\nDeaths: %d\nKDA: %.2f",
		player.PlayerName, player.GamesPlayed, player.Wins, player.MMR, player.Kills, player.Assists, player.Deaths, player.CalculateKda(),
	)
	s.ChannelMessageSend(channelID, stats)
}

// Command to display ELO graph data (for graphing or text output)
func eloGraphCommand(s *discordgo.Session, channelID, playerID string, db *DB) {
	mmrs, timestamps, err := db.GetMmrHistory(playerID)
	if err != nil {
		s.ChannelMessageSend(channelID, fmt.Sprintf("Error fetching MMR history: %v", err))
		return
	}

	message := fmt.Sprintf("MMR History for player %s:\n", playerID)
	for i := range mmrs {
		message += fmt.Sprintf("%s: %d MMR\n", timestamps[i], mmrs[i])
	}
	s.ChannelMessageSend(channelID, message)
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

func handleTeamsCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string, db *DB, discordInstance *Discord) {
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
	playerIDs, err := discordInstance.GetPlayersInVoiceChannel(guildID, voiceChannelID)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Error fetching players in voice channel: %v", err))
		return
	}

	// Remove commentatorID if not taking all
	if !takeAll {
		var filteredIDs []string
		for _, id := range playerIDs {
			if id != commentatorID {
				filteredIDs = append(filteredIDs, id)
			}
		}
		playerIDs = filteredIDs
	}

	// Load players from DB or create new ones
	var players []*Player
	for _, playerID := range playerIDs {
		player, err := db.GetPlayer(playerID)
		if err != nil {
			// If player doesn't exist, create a new one
			if err == sql.ErrNoRows {
				playerName, err := discordInstance.GetPlayerName(playerID)
				if err != nil {
					s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Error getting player name: %v", err))
					return
				}
				player = &Player{
					PlayerID:   playerID,
					PlayerName: playerName,
					MMR:        1000, // Default MMR
				}
				err = db.SavePlayer(player)
				if err != nil {
					s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Error saving player: %v", err))
					return
				}
			} else {
				s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Error getting player: %v", err))
				return
			}
		}
		players = append(players, player)
	}

	// Select teams based on MMR
	team1, team2, err := BalanceTeams(players)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Error balancing teams: %v", err))
		return
	}

	// Store teams in DB
	teamStorage := NewTeamStorage(db, 48*time.Hour)
	err = teamStorage.StoreTeams(team1, team2)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Error storing teams: %v", err))
		return
	}

	// Send team compositions
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Team 1: %v\nTeam 2: %v", getTeamNames(team1.Players), getTeamNames(team2.Players)))
}

//
//func handleWinCommand(s *discordgo.Session, m *discordgo.MessageCreate, match *Match, ts *TeamStorage, args []string) {
//	if len(args) < 2 {
//		s.ChannelMessageSend(m.ChannelID, "Please specify the winning team (team1won or team2won).")
//		return
//	}
//
//	var winningTeam int
//	if args[1] == "team1" {
//		winningTeam = 1
//	} else if args[1] == "team2" {
//		winningTeam = 2
//	} else {
//		s.ChannelMessageSend(m.ChannelID, "Invalid team. Use team1won or team2won.")
//		return
//	}
//
//	// Retrieve stored teams from the database
//	team1, team2, err := ts.GetStoredTeams()
//	if err != nil {
//		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Error: %v. Please run `!teams` to form new teams.", err))
//		return
//	}
//
//	// Save the match result using the stored teams
//	matchID := match.SaveMatch(team1, team2)
//	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Match %d reported: Team %d won!", matchID, winningTeam))
//}
//
//func handleEndSessionCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
//	clearStoredTeamsFromDB()
//	s.ChannelMessageSend(m.ChannelID, "The current match session has ended, and teams have been cleared.")
//}
