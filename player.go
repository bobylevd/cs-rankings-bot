package main

import (
	"database/sql"
	"github.com/bwmarrin/discordgo"
	"log"
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
}

// Retrieve or create a player
func getOrCreatePlayer(playerID string, s *discordgo.Session) *Player {
	var player Player
	err := db.QueryRow("SELECT PlayerID, PlayerName, CoreMember, ELO, GamesPlayed, Wins, Kills, Assists, Deaths FROM players WHERE PlayerID = ?", playerID).Scan(
		&player.PlayerID, &player.PlayerName, &player.CoreMember, &player.ELO, &player.GamesPlayed, &player.Wins, &player.Kills, &player.Assists, &player.Deaths,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			playerName := getPlayerNameFromDiscord(s, playerID)
			player = Player{
				PlayerID:   playerID,
				PlayerName: playerName,
				ELO:        1000, // Starting ELO
			}
			savePlayerStats(&player)
		} else {
			log.Fatal("Error retrieving player:", err)
		}
	}
	return &player
}

func selectPlayersForGame() ([]*Player, error) {
	var corePlayers []*Player
	var nonCorePlayers []*Player

	// Retrieve all players from the database
	rows, err := db.Query("SELECT PlayerID, PlayerName, CoreMember, ELO, GamesPlayed, Wins, Kills, Assists, Deaths FROM players")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Process each row (player) and categorize them into core and non-core players
	for rows.Next() {
		var player Player
		err := rows.Scan(&player.PlayerID, &player.PlayerName, &player.CoreMember, &player.ELO, &player.GamesPlayed, &player.Wins, &player.Kills, &player.Assists, &player.Deaths)
		if err != nil {
			return nil, err
		}

		// Categorize based on CoreMember status
		if player.CoreMember {
			corePlayers = append(corePlayers, &player)
		} else {
			nonCorePlayers = append(nonCorePlayers, &player)
		}
	}

	// Combine core players with non-core players
	finalSelection := append(corePlayers, nonCorePlayers...)

	// Ensure we only return 10 players
	if len(finalSelection) > 10 {
		finalSelection = finalSelection[:10]
	}

	return finalSelection, nil
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
