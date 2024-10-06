package main

import (
	"database/sql"
	"errors"
	"github.com/bwmarrin/discordgo"
	"log"
	"math/rand"
	"strings"
	"time"
)

type Player struct {
	PlayerID    string
	PlayerName  string
	CoreMember  bool
	ELO         int
	GamesPlayed int
	Wins        int
	Kills       int
	Assists     int
	Deaths      int
	Percentile  float64
	KDA         float64
}

// Retrieve or create a player, optionally with a player name
func getOrCreatePlayer(playerID string, playerNameOptional ...string) *Player {
	var player Player
	var playerName string

	// Check if a player name was passed, otherwise fetch from Discord
	if len(playerNameOptional) > 0 {
		playerName = playerNameOptional[0]
	} else {
		playerName = getPlayerNameFromDatabaseOrDefault(playerID)
	}

	err := db.QueryRow("SELECT PlayerID, PlayerName, CoreMember, ELO, GamesPlayed, Wins, Kills, Assists, Deaths FROM players WHERE PlayerID = ?", playerID).Scan(
		&player.PlayerID, &player.PlayerName, &player.CoreMember, &player.ELO, &player.GamesPlayed, &player.Wins, &player.Kills, &player.Assists, &player.Deaths,
	)

	if err == sql.ErrNoRows {
		// Player not found, create a new one with a default or provided name
		player = Player{
			PlayerID:   playerID,
			PlayerName: playerName,
			ELO:        1000, // Starting ELO for new players
		}
		savePlayerStats(&player)
	} else if err != nil {
		log.Fatal("Error retrieving player from database:", err)
	}

	return &player
}

func selectPlayersForGame(playerIDs []string, takeAll bool, commentatorID string) ([]*Player, []*Player, error) {
	var corePlayers []*Player
	var nonCorePlayers []*Player
	var allPlayers []*Player

	if len(playerIDs) == 0 {
		return nil, nil, errors.New("no players provided")
	}

	// Process each player ID, check if they exist in the database or create them
	for _, playerID := range playerIDs {
		// Exclude the commentator from the selection
		if playerID == commentatorID {
			continue
		}

		player := getOrCreatePlayer(playerID)
		allPlayers = append(allPlayers, player)

		// Categorize based on CoreMember status
		if player.CoreMember {
			corePlayers = append(corePlayers, player)
		} else {
			nonCorePlayers = append(nonCorePlayers, player)
		}
	}

	// Scenario 1: Not taking all players, limit to 10
	if !takeAll && len(allPlayers) > 10 {
		finalSelection := selectTopPlayers(corePlayers, nonCorePlayers)
		team1, team2 := splitTeamsByELO(finalSelection)
		return team1, team2, nil
	}

	// Scenario 2: Take all players, balance them by ELO
	team1, team2 := splitTeamsByELO(allPlayers)
	return team1, team2, nil
}

// Helper function to get users currently in a voice channel
func getUsersInVoiceChannel(s *discordgo.Session, guildID, voiceChannelID string) ([]*discordgo.Member, error) {
	guild, err := s.State.Guild(guildID)
	if err != nil {
		return nil, err
	}

	var membersInVoice []*discordgo.Member
	for _, vs := range guild.VoiceStates {
		if vs.ChannelID == voiceChannelID {
			member, err := s.State.Member(guildID, vs.UserID)
			if err == nil {
				membersInVoice = append(membersInVoice, member)
			}
		}
	}
	return membersInVoice, nil
}

// Save or update player stats
func savePlayerStats(player *Player) {
	_, err := db.Exec(`
		INSERT INTO players (PlayerID, PlayerName, CoreMember, ELO, GamesPlayed, Wins, Kills, Assists, Deaths)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(PlayerID) DO UPDATE SET 
			PlayerName=excluded.PlayerName,
			CoreMember=excluded.CoreMember,
			ELO=excluded.ELO,
			GamesPlayed=excluded.GamesPlayed,
			Wins=excluded.Wins,
			Kills=excluded.Kills,
			Assists=excluded.Assists,
			Deaths=excluded.Deaths;
	`, player.PlayerID, player.PlayerName, player.CoreMember, player.ELO, player.GamesPlayed, player.Wins, player.Kills, player.Assists, player.Deaths)
	if err != nil {
		log.Fatal("Error saving player data:", err)
	}
}

func getPlayerStats(playerID string) (*Player, error) {
	var player Player
	err := db.QueryRow("SELECT PlayerID, PlayerName, CoreMember, ELO, GamesPlayed, Wins, Kills, Assists, Deaths FROM players WHERE PlayerID = ?", playerID).Scan(
		&player.PlayerID, &player.PlayerName, &player.CoreMember, &player.ELO, &player.GamesPlayed, &player.Wins, &player.Kills, &player.Assists, &player.Deaths,
	)
	if err != nil {
		return nil, err
	}
	return &player, nil
}

func selectTopPlayers(corePlayers, nonCorePlayers []*Player) []*Player {
	// Shuffle the non-core players to make the selection random
	rand.Seed(time.Now().UnixNano()) // Seed the random number generator
	rand.Shuffle(len(nonCorePlayers), func(i, j int) {
		nonCorePlayers[i], nonCorePlayers[j] = nonCorePlayers[j], nonCorePlayers[i]
	})
	finalSelection := append(corePlayers, nonCorePlayers...)

	// Limit the selection to 10 players if necessary
	if len(finalSelection) > 10 {
		finalSelection = finalSelection[:10]
	}

	return finalSelection
}

// Helper function to get player IDs from a list of players
func getPlayerIDs(players []*Player) string {
	var ids []string
	for _, player := range players {
		ids = append(ids, player.PlayerID)
	}
	return strings.Join(ids, ",")
}

// Convert a comma-separated string of player IDs into a slice of Player structs
func getPlayersFromIDs(playerIDs string) []*Player {
	var players []*Player
	ids := strings.Split(playerIDs, ",")

	for _, id := range ids {
		var player Player
		err := db.QueryRow("SELECT PlayerID, PlayerName, CoreMember, ELO, GamesPlayed, Wins, Kills, Assists, Deaths FROM players WHERE PlayerID = ?", id).Scan(
			&player.PlayerID, &player.PlayerName, &player.CoreMember, &player.ELO, &player.GamesPlayed, &player.Wins, &player.Kills, &player.Assists, &player.Deaths,
		)
		if err != nil {
			log.Printf("Error retrieving player with ID %s: %v", id, err)
			continue // Skip players that couldn't be retrieved
		}
		players = append(players, &player)
	}

	return players
}

// Helper function to map player IDs to their stats
func mapPlayersToStats(winners []*Player, stat string, losers []*Player) map[string]int {
	stats := make(map[string]int)

	for _, player := range winners {
		switch stat {
		case "kills":
			stats[player.PlayerID] = player.Kills
		case "assists":
			stats[player.PlayerID] = player.Assists
		case "deaths":
			stats[player.PlayerID] = player.Deaths
		}
	}

	for _, player := range losers {
		switch stat {
		case "kills":
			stats[player.PlayerID] = player.Kills
		case "assists":
			stats[player.PlayerID] = player.Assists
		case "deaths":
			stats[player.PlayerID] = player.Deaths
		}
	}

	return stats
}

// GetPlayerIDsFromVoiceChannel retrieves the list of player IDs from a Discord voice channel
func GetPlayerIDsFromVoiceChannel(s *discordgo.Session, guildID string, voiceChannelID string) ([]string, error) {
	voiceChannelMembers, err := getUsersInVoiceChannel(s, guildID, voiceChannelID)
	if err != nil {
		return nil, err
	}

	var playerIDs []string
	for _, member := range voiceChannelMembers {
		playerIDs = append(playerIDs, member.User.ID)
	}

	return playerIDs, nil
}

// Helper function to get player name either from the database or return a default
func getPlayerNameFromDatabaseOrDefault(playerID string) string {
	var playerName string
	err := db.QueryRow("SELECT PlayerName FROM players WHERE PlayerID = ?", playerID).Scan(&playerName)
	if err == sql.ErrNoRows {
		return "UnknownPlayer" // Default name if none is found
	} else if err != nil {
		log.Printf("Error retrieving player name for %s: %v", playerID, err)
		return "UnknownPlayer"
	}
	return playerName
}
