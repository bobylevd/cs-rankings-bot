package main

import (
	"encoding/json"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"io/ioutil"
	"log"
	"os"
	"strings"
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

			// Extract winner and looser teams
			winners := match["winner"]
			losers := match["looser"]

			// Process match data
			processMatchData(winners, losers, s)
		}
	}
}

// Process a single match, updating stats and calculating ELO
func processMatchData(winners, losers map[string]map[string]int, s *discordgo.Session) {
	var winnerPlayers []*Player
	var loserPlayers []*Player

	// Process winners
	for playerID, stats := range winners {
		player := getOrCreatePlayer(playerID, s) // Get or create player
		player.GamesPlayed++
		player.Wins++
		player.Kills += stats["kills"]
		player.Assists += stats["assists"]
		player.Deaths += stats["deaths"]
		savePlayerStats(player) // Save updated player stats
		winnerPlayers = append(winnerPlayers, player)
	}

	// Process losers
	for playerID, stats := range losers {
		player := getOrCreatePlayer(playerID, s) // Get or create player
		player.GamesPlayed++
		player.Kills += stats["kills"]
		player.Assists += stats["assists"]
		player.Deaths += stats["deaths"]
		savePlayerStats(player) // Save updated player stats
		loserPlayers = append(loserPlayers, player)
	}

	// Update ELO for both teams
	matchID := saveMatchData(winnerPlayers, loserPlayers) // Save match and get match ID
	updateELO(winnerPlayers, loserPlayers, matchID)       // Update ELO based on the match result
}

// Save match data in the database and return the match ID
func saveMatchData(winners, losers []*Player) int {
	winnerIDs := getPlayerIDs(winners)
	loserIDs := getPlayerIDs(losers)

	kills, _ := json.Marshal(mapPlayersToStats(winners, "kills", losers))
	assists, _ := json.Marshal(mapPlayersToStats(winners, "assists", losers))
	deaths, _ := json.Marshal(mapPlayersToStats(winners, "deaths", losers))

	// Insert match data into the database
	result, err := db.Exec(`
		INSERT INTO matches (Team1, Team2, WinningTeam, Kills, Assists, Deaths)
		VALUES (?, ?, ?, ?, ?, ?)
	`, winnerIDs, loserIDs, 1, kills, assists, deaths)
	if err != nil {
		log.Fatal("Error saving match data:", err)
	}

	// Get the inserted match ID
	matchID, _ := result.LastInsertId()
	return int(matchID)
}

// Helper function to get player IDs from a list of players
func getPlayerIDs(players []*Player) string {
	var ids []string
	for _, player := range players {
		ids = append(ids, player.PlayerID)
	}
	return strings.Join(ids, ",")
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
