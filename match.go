package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"io/ioutil"
	"log"
	"os"
	"time"
)

// Import historical data from a JSON file
func importHistoricalData(filename string, s *discordgo.Session) {
	// Open and read the JSON file
	jsonFile, err := os.Open(filename)
	if err != nil {
		log.Fatal("Error opening JSON file:", err)
	}
	defer jsonFile.Close()

	// Parse JSON data
	byteValue, _ := ioutil.ReadAll(jsonFile)
	var matchData map[string]map[string]map[string]map[string]map[string]int
	err = json.Unmarshal(byteValue, &matchData)
	if err != nil {
		log.Fatal("Error parsing JSON data:", err)
	}

	// Process each day and game in the JSON data
	for day, games := range matchData {
		fmt.Printf("Processing matches for day: %s\n", day)
		for gameID, match := range games {
			fmt.Printf("Processing match: %s\n", gameID)

			// Extract winner and loser teams
			winners := match["winner"]
			losers := match["looser"]

			// Convert winner and loser data into the correct format
			team1 := winners // Team 1 is the winner
			team2 := losers  // Team 2 is the loser
			winningTeam := 1 // Since team1 is the winner in historical data

			// Process the match data (team1 = winners, team2 = losers)
			processMatchData(team1, team2, winningTeam, s)
		}
	}
}

// Process a single match, updating stats and calculating ELO
func processMatchData(team1, team2 map[string]map[string]int, winningTeam int, s *discordgo.Session) {
	var team1Players []*Player
	var team2Players []*Player

	// Process Team 1 players
	for playerID, stats := range team1 {
		player := getOrCreatePlayer(playerID) // Get or create player
		player.GamesPlayed++
		if winningTeam == 1 {
			player.Wins++ // Only increment wins if Team 1 won
		}
		player.Kills += stats["kills"]
		player.Assists += stats["assists"]
		player.Deaths += stats["deaths"]
		savePlayerStats(player) // Save updated player stats
		team1Players = append(team1Players, player)
	}

	// Process Team 2 players
	for playerID, stats := range team2 {
		player := getOrCreatePlayer(playerID) // Get or create player
		player.GamesPlayed++
		if winningTeam == 2 {
			player.Wins++ // Only increment wins if Team 2 won
		}
		player.Kills += stats["kills"]
		player.Assists += stats["assists"]
		player.Deaths += stats["deaths"]
		savePlayerStats(player) // Save updated player stats
		team2Players = append(team2Players, player)
	}

	// Delegate saving match result and stats to saveMatchData
	matchID := saveMatchData(team1Players, team2Players, winningTeam) // Save match and get match ID

	// Update ELO for both teams
	updateELO(team1Players, team2Players, matchID) // Update ELO based on the match result
}

// Save match data in the database and return the match ID
func saveMatchData(team1, team2 []*Player, winningTeam int) int {
	team1IDs := getPlayerIDs(team1)
	team2IDs := getPlayerIDs(team2)

	// Assuming player stats are already available and passed into the function
	kills, _ := json.Marshal(mapPlayersToStats(team1, "kills", team2))
	assists, _ := json.Marshal(mapPlayersToStats(team1, "assists", team2))
	deaths, _ := json.Marshal(mapPlayersToStats(team1, "deaths", team2))

	// Insert match data into the database
	result, err := db.Exec(`
		INSERT INTO matches (Team1, Team2, WinningTeam, Kills, Assists, Deaths)
		VALUES (?, ?, ?, ?, ?, ?)
	`, team1IDs, team2IDs, winningTeam, kills, assists, deaths)
	if err != nil {
		log.Fatal("Error saving match data:", err)
	}

	// Get the inserted match ID
	matchID, _ := result.LastInsertId()
	return int(matchID)
}

// Store teams in the database with a timestamp
func storeTeamsInDB(team1, team2 []*Player) {
	team1IDs := getPlayerIDs(team1)
	team2IDs := getPlayerIDs(team2)

	_, err := db.Exec(`
		INSERT INTO temp_teams (id, team1, team2, timestamp)
		VALUES (1, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET team1 = ?, team2 = ?, timestamp = CURRENT_TIMESTAMP;
	`, team1IDs, team2IDs, team1IDs, team2IDs)
	if err != nil {
		log.Fatal("Error storing teams in database:", err)
	}
}

func getStoredTeamsFromDB() ([]*Player, []*Player, error) {
	var team1IDs, team2IDs string
	var timestamp time.Time

	// Query to get teams and their timestamp
	err := db.QueryRow("SELECT team1, team2, timestamp FROM temp_teams WHERE id = 1").Scan(&team1IDs, &team2IDs, &timestamp)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, errors.New("no stored teams found")
		}
		return nil, nil, err
	}

	// Check if the timestamp is older than 48 hours
	if time.Since(timestamp).Hours() > 48 {
		// Clear old teams and notify the caller
		clearStoredTeamsFromDB()
		return nil, nil, errors.New("stored teams have expired")
	}

	// If the teams are still valid, return them
	team1 := getPlayersFromIDs(team1IDs)
	team2 := getPlayersFromIDs(team2IDs)
	return team1, team2, nil
}

// Clear stored teams after reporting or expiration
func clearStoredTeamsFromDB() {
	_, err := db.Exec("DELETE FROM temp_teams WHERE id = 1")
	if err != nil {
		log.Fatal("Error clearing stored teams from database:", err)
	}
}
