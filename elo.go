package main

import (
	"github.com/bwmarrin/discordgo"
	"log"
	"math"
	"sort"
)

// Record ELO change for a player
func recordELOChange(player *Player, matchID int) {
	_, err := db.Exec(`
		INSERT INTO elo_history (PlayerID, ELO, MatchID)
		VALUES (?, ?, ?)
	`, player.PlayerID, player.ELO, matchID)
	if err != nil {
		log.Fatal("Error saving ELO change:", err)
	}
}

// Calculate K-factor dynamically based on player stats
func calculateKFactor(player *Player) int {
	if player.GamesPlayed < 5 {
		return 40 // High K-factor for new players
	} else if player.GamesPlayed > 15 {
		return 20 // Medium K-factor for experienced players
	} else if player.ELO > 2000 {
		return 10 // Low K-factor for high-rated players
	} else {
		return 32 // Default K-factor for most players
	}
}

// Update ELO based on match result
func updateELO(winners, losers []*Player, matchID int) {
	eloTeam1 := calculateTeamELO(winners)
	eloTeam2 := calculateTeamELO(losers)

	for _, player := range winners {
		kFactor := calculateKFactor(player)
		expectedScore := 1 / (1 + math.Pow(10, float64(eloTeam2-eloTeam1)/400))
		player.ELO += int(float64(kFactor) * (1 - expectedScore))
		savePlayerStats(player)
		recordELOChange(player, matchID)
	}

	for _, player := range losers {
		kFactor := calculateKFactor(player)
		expectedScore := 1 / (1 + math.Pow(10, float64(eloTeam1-eloTeam2)/400))
		player.ELO += int(float64(kFactor) * (0 - expectedScore))
		savePlayerStats(player)
		recordELOChange(player, matchID)
	}
}

// Calculate team ELO (average)
func calculateTeamELO(players []*Player) int {
	totalELO := 0
	for _, player := range players {
		totalELO += player.ELO
	}
	return totalELO / len(players)
}

func splitTeamsByELO(players []*Player) ([]*Player, []*Player) {
	// Sort players by ELO in descending order
	sort.Slice(players, func(i, j int) bool {
		return players[i].ELO > players[j].ELO
	})

	var team1 []*Player
	var team2 []*Player
	var eloTeam1, eloTeam2 int

	// Greedily assign players to teams based on ELO
	for _, player := range players {
		if eloTeam1 <= eloTeam2 {
			team1 = append(team1, player)
			eloTeam1 += player.ELO
		} else {
			team2 = append(team2, player)
			eloTeam2 += player.ELO
		}
	}

	return team1, team2
}

func getPlayerELOHistory(playerID string) ([]int, []string, error) {
	rows, err := db.Query(`
		SELECT ELO, Timestamp FROM elo_history WHERE PlayerID = ? ORDER BY Timestamp ASC
	`, playerID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var elos []int
	var timestamps []string
	for rows.Next() {
		var elo int
		var timestamp string
		err := rows.Scan(&elo, &timestamp)
		if err != nil {
			return nil, nil, err
		}
		elos = append(elos, elo)
		timestamps = append(timestamps, timestamp)
	}
	return elos, timestamps, nil
}

func getPlayerNameFromDiscord(s *discordgo.Session, playerID string) string {
	user, err := s.User(playerID)
	if err != nil {
		log.Printf("Error fetching user for ID %s: %v", playerID, err)
		return "Unknown" // Fallback name
	}
	return user.Username
}
