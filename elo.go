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

func calculateTeamKDA(players []*Player) float64 {
	totalKDA := 0.0
	for _, player := range players {
		kda := float64(player.Kills+player.Assists) / math.Max(1.0, float64(player.Deaths))
		totalKDA += kda
	}
	return totalKDA / float64(len(players))
}

func calculateContextualELOAdjustment(player *Player, teamKDAAvg, playerKDA, expectedScore, actualScore, kdaFactor float64, kFactor int) int {
	// Calculate base ELO adjustment
	eloChange := float64(kFactor) * kdaFactor * (actualScore - expectedScore)

	// If the player's KDA is better than the team's average, mitigate loss or boost gain
	if playerKDA > teamKDAAvg {
		eloChange = math.Max(eloChange, 0) // Ensure no loss for top performers on losing team
		if actualScore == 0.0 {
			eloChange = math.Max(eloChange, 5) // Provide small gain if they outperformed
		}
	}

	// Cap ELO loss for high-KDA players (e.g., KDA > 2.0)
	if actualScore == 0.0 && playerKDA > 1.38 { // 75th percentile KDA is 1.37
		eloChange = math.Max(eloChange, -5) // Cap the loss to avoid over-penalizing
	}

	return int(eloChange)
}

// Calculate K-factor dynamically based on player stats
func calculateKFactor(player *Player) int {
	if player.GamesPlayed < 3 {
		return 40 // High K-factor for new players
	} else if player.GamesPlayed > 10 {
		return 20 // Medium K-factor for experienced players
	} else if player.ELO > 1300 {
		return 10 // Low K-factor for high-rated players
	} else {
		return 32 // Default K-factor for most players
	}
}

// Function to calculate KDA factor based on games played
func calculateKDAFactor(player *Player, kda float64) float64 {
	if player.GamesPlayed < 5 {
		// Reduce KDA impact for players with fewer than 5 games
		return math.Max(0.5, kda/4) // Cap the impact to avoid large changes
	} else if player.GamesPlayed <= 10 {
		// Partially reduce KDA impact for players with 5-10 games
		return math.Min(kda/2, 1.2) // Allow some influence but limit it
	} else {
		// Full KDA impact for players with more than 10 games
		return math.Min(kda/2, 1.5) // Regular KDA boost
	}
}

// Update ELO based on match result, incorporating KDA influence and win/loss reduction
func updateELO(winners, losers []*Player, matchID int) {
	eloTeam1 := calculateTeamELO(winners)
	eloTeam2 := calculateTeamELO(losers)

	// Calculate the average KDA for each team
	team1KDAAvg := calculateTeamKDA(winners)
	team2KDAAvg := calculateTeamKDA(losers)

	// Process winners
	for _, player := range winners {
		kFactor := calculateKFactor(player)
		expectedScore := 1 / (1 + math.Pow(10, float64(eloTeam2-eloTeam1)/400))
		actualScore := 1.0 // Win

		// Calculate player's KDA score
		playerKDA := float64(player.Kills+player.Assists) / math.Max(1.0, float64(player.Deaths))

		// Adjust ELO using contextual KDA adjustment
		kdaFactor := calculateKDAFactor(player, playerKDA)
		eloChange := calculateContextualELOAdjustment(player, team1KDAAvg, playerKDA, expectedScore, actualScore, kdaFactor, kFactor)

		// Apply ELO change
		player.ELO += eloChange

		// Save player stats and record ELO change
		savePlayerStats(player)
		recordELOChange(player, matchID)
	}

	// Process losers
	for _, player := range losers {
		kFactor := calculateKFactor(player)
		expectedScore := 1 / (1 + math.Pow(10, float64(eloTeam1-eloTeam2)/400))
		actualScore := 0.0 // Loss

		// Calculate player's KDA score
		playerKDA := float64(player.Kills+player.Assists) / math.Max(1.0, float64(player.Deaths))

		// Adjust ELO using contextual KDA adjustment
		kdaFactor := calculateKDAFactor(player, playerKDA)
		eloChange := calculateContextualELOAdjustment(player, team2KDAAvg, playerKDA, expectedScore, actualScore, kdaFactor, kFactor)

		// If the player performed well (high KDA), mitigate ELO loss
		if playerKDA > team2KDAAvg {
			eloChange = int(math.Max(float64(eloChange), -5)) // Limit loss for strong performers
		}

		// Apply ELO change
		player.ELO += eloChange

		// Save player stats and record ELO change
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
