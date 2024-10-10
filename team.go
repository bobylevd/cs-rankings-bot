package main

import (
	"math/rand"
	"sort"
	"strings"
	"time"
)

type Team struct {
	Name    string
	Winner  bool
	Players []*Player
}

func (t *Team) GetPlayerIDs() string {
	var ids []string
	for _, player := range t.Players {
		ids = append(ids, player.PlayerID)
	}
	return strings.Join(ids, ",")
}

// Team method to calculate average team Mmr
func (t *Team) calculateTeamMmr() int {
	totalMMR := 0
	for _, player := range t.Players {
		totalMMR += player.MMR
	}
	return totalMMR / len(t.Players)
}

// Team method to calculate average team KDA
func (t *Team) calculateTeamKDA() float64 {
	totalKDA := 0.0
	for _, player := range t.Players {
		playerKDA := player.CalculateKda()
		totalKDA += playerKDA
	}
	return totalKDA / float64(len(t.Players))
}

func BalanceTeams(players []*Player) (*Team, *Team, error) {
	// Simple balancing algorithm based on MMR
	sort.Slice(players, func(i, j int) bool {
		return players[i].MMR > players[j].MMR
	})

	team1 := &Team{Name: "Team 1", Players: []*Player{}}
	team2 := &Team{Name: "Team 2", Players: []*Player{}}

	for i, player := range players {
		if i%2 == 0 {
			team1.Players = append(team1.Players, player)
		} else {
			team2.Players = append(team2.Players, player)
		}
	}

	// Shuffle teams to add some randomness
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(team1.Players), func(i, j int) {
		team1.Players[i], team1.Players[j] = team1.Players[j], team1.Players[i]
	})
	rand.Shuffle(len(team2.Players), func(i, j int) {
		team2.Players[i], team2.Players[j] = team2.Players[j], team2.Players[i]
	})

	return team1, team2, nil
}

func getTeamNames(team *Team) string {
	var names []string
	for _, player := range team.Players {
		names = append(names, player.PlayerName)
	}
	return strings.Join(names, ", ")
}
