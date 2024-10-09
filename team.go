package main

import (
	"strings"
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
