package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Match struct {
	db          *DB
	Winner      *Team
	Loser       *Team
	WinningTeam int
	MatchID     int
}

// Save the match and update player stats
func (m *Match) SaveMatch(db *DB) (int, error) {
	// Save the match to the database
	matchID, err := db.SaveMatch(m)
	if err != nil {
		return 0, err
	}
	m.MatchID = matchID

	// Update MMR for players
	updateMmr(m)

	// Save player stats to the database
	err = savePlayerStats(append(m.Winner.Players, m.Loser.Players...), db)
	if err != nil {
		return matchID, err
	}

	return matchID, nil
}

// Store teams temporarily in the database
func (m *Match) StoreTeams(db *DB) error {
	return db.StoreTeams(m.Winner, m.Loser)
}

// Prepare stats for the match (e.g., for displaying or saving)
func (m *Match) PrepareStats() (string, string, string, error) {
	// Calculate stats for both teams
	kills, err := json.Marshal(mapTeamsToStats(m.Winner, "kills", m.Loser))
	if err != nil {
		return "", "", "", err
	}

	assists, err := json.Marshal(mapTeamsToStats(m.Winner, "assists", m.Loser))
	if err != nil {
		return "", "", "", err
	}

	deaths, err := json.Marshal(mapTeamsToStats(m.Winner, "deaths", m.Loser))
	if err != nil {
		return "", "", "", err
	}

	return string(kills), string(assists), string(deaths), nil
}

// mapTeamsToStats maps player names to their stats for both teams
func mapTeamsToStats(team1 *Team, stat string, team2 *Team) map[string]int {
	stats := make(map[string]int)
	for _, player := range team1.Players {
		stats[player.PlayerName] = getPlayerStat(player, stat)
	}
	for _, player := range team2.Players {
		stats[player.PlayerName] = getPlayerStat(player, stat)
	}
	return stats
}

// getPlayerStat retrieves the specified stat from a player
func getPlayerStat(player *Player, stat string) int {
	switch stat {
	case "kills":
		return player.Kills
	case "assists":
		return player.Assists
	case "deaths":
		return player.Deaths
	default:
		return 0
	}
}

// Helper function to get players from IDs
func getPlayersFromIDs(ids string, db *DB) ([]*Player, error) {
	playerIDs := strings.Split(ids, ",")
	players := []*Player{}
	for _, id := range playerIDs {
		player, err := db.GetPlayer(id)
		if err != nil {
			return nil, fmt.Errorf("failed to get player %s: %v", id, err)
		}
		players = append(players, player)
	}
	return players, nil
}
