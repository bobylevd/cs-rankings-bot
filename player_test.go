package main

import (
	"log"
	"math/rand"
	"testing"
	"time"
)

// Select 10 random players from the full player list
func getRandomPlayers(playerIDs []string, num int) []string {
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(playerIDs), func(i, j int) {
		playerIDs[i], playerIDs[j] = playerIDs[j], playerIDs[i]
	})

	if len(playerIDs) > num {
		playerIDs = playerIDs[:num]
	}
	return playerIDs
}

func TestSelectPlayersForGameWithRandomRealPlayers(t *testing.T) {
	initDB()

	// Get all players from the database
	allPlayerIDs, err := getPlayersFromDB()
	if err != nil {
		t.Fatalf("Error fetching players from DB: %v", err)
	}

	// Pick 10 random players
	playerIDs := getRandomPlayers(allPlayerIDs, 10)

	commentatorID := "108220450194092032" // Assuming this is your commentator ID

	// Call the function to select players for the game
	team1, team2, err := selectPlayersForGame(playerIDs, true, commentatorID)
	if err != nil {
		t.Fatalf("Error selecting players: %v", err)
	}

	// Check if the teams were created correctly
	if len(team1) == 0 || len(team2) == 0 {
		t.Fatalf("Teams should not be empty. Team1: %d, Team2: %d", len(team1), len(team2))
	}

	// Ensure the number of players in both teams adds up to 10
	if len(team1)+len(team2) != 10 {
		t.Fatalf("Expected 10 players but got %d in total", len(team1)+len(team2))
	}

	// Log the names of players in each team
	logTeamNames("Team 1", team1)
	logTeamNames("Team 2", team2)

	// Optional: Verify that ELO differences between teams are minimized
	team1ELO := calculateTeamELO(team1)
	team2ELO := calculateTeamELO(team2)

	eloDiff := abs(team1ELO - team2ELO)
	if eloDiff > 100 {
		t.Fatalf("ELO difference between teams is too high: %d", eloDiff)
	}

	t.Logf("Teams balanced successfully. Team 1 ELO: %d, Team 2 ELO: %d", team1ELO, team2ELO)
}

// Helper function to log player names for a team
func logTeamNames(teamName string, team []*Player) {
	log.Printf("%s:", teamName)
	for _, player := range team {
		log.Printf("Player: %s (ELO: %d)", player.PlayerName, player.ELO)
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
