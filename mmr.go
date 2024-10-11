package main

import (
	"log"
	"math"
)

// Mmr update process using the Match, Team, and Player entities
func updateMmr(match *Match) {
	// Calculate MMR changes for both the winner and loser teams
	calculateMMRChanges(match)

	// Record the MMR changes for each player in both teams
	for _, player := range match.Winner.Players {
		recordMmrChange(player, match.MatchID, match.db)
	}
	for _, player := range match.Loser.Players {
		recordMmrChange(player, match.MatchID, match.db)
	}
}

// savePlayerStats is responsible for saving player data after MMR calculation
func savePlayerStats(players []*Player, db *DB) error {
	for _, player := range players {
		// Save player stats using the DB reference
		if err := db.SavePlayer(player); err != nil {
			log.Printf("Error saving stats for player %s: %v", player.PlayerID, err)
			return err
		}
	}
	return nil
}

// Calculate MMR changes for both teams based on match result
func calculateMMRChanges(match *Match) {
	team1MMR := match.Winner.calculateTeamMmr()
	team2MMR := match.Loser.calculateTeamMmr()
	team1KDAAvg := match.Winner.calculateTeamKDA()
	team2KDAAvg := match.Loser.calculateTeamKDA()

	// Process winners (actualScore = 1.0 for winners)
	for _, player := range match.Winner.Players {
		kFactor := calculateKFactor(player)
		expectedScore := calculateExpectedScore(team2MMR, team1MMR)
		playerKDA := player.CalculateKda()
		kdaFactor := calculateKDAFactor(player, playerKDA)

		MMRChange := calculateContextualMmrAdjustment(team1KDAAvg, playerKDA, expectedScore, 1.0, kdaFactor, kFactor)
		player.MMR += MMRChange
		player.Wins++
		player.GamesPlayed++
	}

	// Process losers (actualScore = 0.0 for losers)
	for _, player := range match.Loser.Players {
		kFactor := calculateKFactor(player)
		expectedScore := calculateExpectedScore(team1MMR, team2MMR)
		playerKDA := player.CalculateKda()
		kdaFactor := calculateKDAFactor(player, playerKDA)

		MMRChange := calculateContextualMmrAdjustment(team2KDAAvg, playerKDA, expectedScore, 0.0, kdaFactor, kFactor)
		player.MMR += MMRChange
		player.GamesPlayed++
	}
}

// Calculate expected score based on MMR difference between teams
func calculateExpectedScore(teamMmr, opponentMmr int) float64 {
	return 1.0 / (1.0 + math.Pow(10, float64(opponentMmr-teamMmr)/400))
}

// Calculate MMR and KDA adjustments for each player based on team performance
func calculateContextualMmrAdjustment(teamKDAAvg, playerKDA, expectedScore, actualScore, kdaFactor float64, kFactor int) int {
	MMRChange := float64(kFactor) * kdaFactor * (actualScore - expectedScore)

	// If the player outperforms the team, mitigate the loss or boost the gain
	if playerKDA > teamKDAAvg {
		MMRChange = math.Max(MMRChange, 0) // Ensure no loss for top performers
		if actualScore == 0.0 {
			MMRChange = math.Max(MMRChange, 5) // Small gain even on loss
		}
	}

	// Cap MMR loss for high-performing players
	if actualScore == 0.0 && playerKDA > 1.5 {
		MMRChange = math.Max(MMRChange, -5) // Cap the loss for strong players
	}

	return int(MMRChange)
}

// Record MMR change for a player in the history table
func recordMmrChange(player *Player, matchID int, db *DB) {
	err := db.RecordMmrHistory(player.PlayerID, player.MMR, matchID)
	if err != nil {
		log.Printf("Error recording MMR history for player %s: %v", player.PlayerID, err)
	}
}

// Calculate K-factor dynamically based on player stats
func calculateKFactor(player *Player) int {
	switch {
	case player.GamesPlayed < 3:
		return 40
	case player.GamesPlayed > 10:
		return 20
	case player.MMR > 1300:
		return 10
	default:
		return 32
	}
}

// Function to calculate KDA factor based on games played
func calculateKDAFactor(player *Player, kda float64) float64 {
	switch {
	case player.GamesPlayed < 5:
		return math.Max(0.5, kda/4)
	case player.GamesPlayed <= 10:
		return math.Min(kda/2, 1.2)
	default:
		return math.Min(kda/2, 1.5)
	}
}
