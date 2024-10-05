package store

import (
	"fmt"
	"math"
	"strings"
)

// Player represents a CS2 player.
type Player struct {
	SteamID    string `db:"steam_id"`
	DiscordID  string `db:"discord_id"`
	CoreMember bool   `db:"core_member"`
	Stat
}

// DiscordRef returns the Discord reference of the player.
func (p Player) DiscordRef() string {
	return fmt.Sprintf("<@%s>", p.DiscordID)
}

// Match represents a CS2 match.
type Match [2]Team

// String returns the string representation of the match in format
// of "<team1> vs <team2>".
func (m Match) String() string {
	return fmt.Sprintf("%s vs %s", m[0], m[1])
}

// UpdateELO returns an updated ELO rating for each player.
func (m Match) UpdateELO(lhsWin bool) []Player {
	winners, losers := m[0], m[1]
	if !lhsWin {
		winners, losers = losers, winners
	}

	winExpected := 1 / (1 + math.Pow(10, float64(winners.ELO()-losers.ELO())/400))

	var result []Player
	for _, player := range winners {
		player.ELO += float64(player.KFactor()) * (1 - winExpected)
		result = append(result, player)
	}

	for _, player := range losers {
		player.ELO += float64(player.KFactor()) * (winExpected - 1)
		result = append(result, player)
	}

	return result
}

// Stat defines a statistic for a player.
type Stat struct {
	ELO         float64 `db:"elo"`
	GamesPlayed int     `db:"games_played"`
	Wins        int     `db:"wins"`
	Kills       int     `db:"kills"`
	Assists     int     `db:"assists"`
	Deaths      int     `db:"deaths"`
}

// Team represents a team of CS2 players.
type Team [5]Player

// String name returns the team members separated by commas.
func (t Team) String() string {
	names := make([]string, 5)
	for i, p := range t {
		names[i] = p.DiscordRef()
	}
	return strings.Join(names, ", ")
}

// ELO returns the total ELO of the team.
func (t Team) ELO() float64 {
	sum := float64(0)
	for _, p := range t {
		sum += p.ELO
	}
	return float64(sum) / 5
}

// KFactor returns the K-factor for the player.
func (s Stat) KFactor() int {
	switch {
	case s.GamesPlayed < 5:
		return 40 // high K-factor for new players
	case s.GamesPlayed > 15:
		return 20 // medium K-factor for experienced players
	case s.ELO > 2000:
		return 10 // low K-factor for high-rated players
	default:
		return 32 // default K-factor for most players
	}
}
