package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

type HistoricalData map[string]map[string]MatchData

type MatchData struct {
	Winner map[string]PlayerStats `json:"winner"`
	Loser  map[string]PlayerStats `json:"loser"`
}

type PlayerStats struct {
	Kills   int `json:"kills"`
	Assists int `json:"assists"`
	Deaths  int `json:"deaths"`
}

// Import historical data from a JSON file
func ImportHistoricalData(filename string, db *DB, discord *Discord) {
	// Open and read the JSON file
	jsonFile, err := os.Open(filename)
	if err != nil {
		log.Fatal("Error opening JSON file:", err)
	}
	defer jsonFile.Close()

	// Parse JSON data
	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		log.Fatal("Error reading JSON file:", err)
	}

	var historicalData HistoricalData
	err = json.Unmarshal(byteValue, &historicalData)
	if err != nil {
		log.Fatal("Error parsing JSON data:", err)
	}

	// Process each day and game in the JSON data
	for day, games := range historicalData {
		fmt.Printf("Processing matches for day: %s\n", day)
		for gameID, match := range games {
			fmt.Printf("Processing match: %s\n", gameID)

			// Process the match data
			err := processHistoricalMatchData(match, db, discord)
			if err != nil {
				log.Printf("Error processing match %s: %v", gameID, err)
			}
		}
	}
}

// Process each match and save it to the database
func processHistoricalMatchData(match MatchData, db *DB, discord *Discord) error {
	// Create Winner Team
	winnerTeam := &Team{
		Name:    "Winner",
		Players: []*Player{},
	}

	// Create Loser Team
	loserTeam := &Team{
		Name:    "Loser",
		Players: []*Player{},
	}

	// Process winner players
	for playerID, stats := range match.Winner {
		player, err := getOrCreatePlayer(playerID, db, discord)
		if err != nil {
			return fmt.Errorf("error retrieving or creating player %s: %v", playerID, err)
		}

		// Update player stats
		player.Kills += stats.Kills
		player.Assists += stats.Assists
		player.Deaths += stats.Deaths
		winnerTeam.Players = append(winnerTeam.Players, player)
	}

	// Process loser players
	for playerID, stats := range match.Loser {
		player, err := getOrCreatePlayer(playerID, db, discord)
		if err != nil {
			return fmt.Errorf("error retrieving or creating player %s: %v", playerID, err)
		}

		// Update player stats
		player.Kills += stats.Kills
		player.Assists += stats.Assists
		player.Deaths += stats.Deaths

		loserTeam.Players = append(loserTeam.Players, player)
	}

	// Create Match instance
	matchInstance := &Match{
		Winner: winnerTeam,
		Loser:  loserTeam,
		db:     db,
	}

	// Save the match and update MMR
	_, err := matchInstance.SaveMatch(db)
	if err != nil {
		return fmt.Errorf("error saving match: %v", err)
	}

	return nil
}

// Helper function to get or create a player
func getOrCreatePlayer(playerID string, db *DB, discord *Discord) (*Player, error) {
	// Try to get the player from the database
	player, err := db.GetPlayer(playerID)
	if err != nil {
		if err == sql.ErrNoRows {
			// Player does not exist, create a new one
			playerName, err := discord.GetPlayerName(playerID)
			if err != nil {
				// Handle error (e.g., user not found on Discord)
				log.Printf("Error retrieving Discord user for playerID %s: %v", playerID, err)
				// Decide whether to continue or return an error
				playerName = playerID // Fallback to playerID if username is not found
			}

			player = &Player{
				PlayerID:    playerID,
				PlayerName:  playerName,
				CoreMember:  false, // Adjust as necessary
				MMR:         1000,  // Default starting MMR
				GamesPlayed: 0,
				Wins:        0,
				Kills:       0,
				Assists:     0,
				Deaths:      0,
				Sniper:      false,
			}
			// Save the new player to the database
			err = db.SavePlayer(player)
			if err != nil {
				return nil, fmt.Errorf("error saving new player %s: %v", playerID, err)
			}
		} else {
			return nil, fmt.Errorf("error retrieving player %s: %v", playerID, err)
		}
	} else {
		// Player exists, update the name from Discord to keep it current
		playerName, err := discord.GetPlayerName(playerID)
		if err != nil {
			log.Printf("Error retrieving Discord user for playerID %s: %v", playerID, err)
		} else {
			player.PlayerName = playerName
			// Optionally, save the updated name to the database
			err = db.SavePlayer(player)
			if err != nil {
				return nil, fmt.Errorf("error updating player %s: %v", playerID, err)
			}
		}
	}
	return player, nil
}
