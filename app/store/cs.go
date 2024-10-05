package store

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/syohex/go-texttable"
)

// DiscordID represents a Discord user ID.
type DiscordID string

// String returns the string representation of the Discord ID.
// String returns the string representation of the Discord ID.
func (d DiscordID) String() string {
	return fmt.Sprintf("<@%s>", string(d))
}

// Player represents a CS2 player.
type Player struct {
	SteamID    string    `db:"steam_id"`
	DiscordID  DiscordID `db:"discord_id"`
	CoreMember bool      `db:"core_member"`
	Stat
}

// InfoTable returns a plain-text table with the player's statistics.
func (p Player) InfoTable() string {
	tbl := &texttable.TextTable{}
	_ = tbl.SetHeader(
		"SteamID",
		"CoreMember",
		"ELO",
		"Games",
		"Wins",
		"WinRate",
		"Kills",
		"Assists",
		"Deaths",
		"KDA",
	)
	_ = tbl.AddRow(
		p.SteamID,
		strconv.FormatBool(p.CoreMember),
		fmt.Sprintf("%.2f", p.ELO),
		strconv.Itoa(p.GamesPlayed),
		strconv.Itoa(p.Wins),
		fmt.Sprintf("%.2f%%", p.WinRate()*100),
		strconv.Itoa(p.Kills),
		strconv.Itoa(p.Assists),
		strconv.Itoa(p.Deaths),
		fmt.Sprintf("%.2f", p.KDA()),
	)
	return tbl.Draw()
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

// WinRate returns the win rate of the player.
func (s Stat) WinRate() float64 {
	if s.GamesPlayed == 0 {
		return 0
	}
	return float64(s.Wins) / float64(s.GamesPlayed)
}

// KDA returns the kill-death-assist ratio of the player.
func (s Stat) KDA() float64 {
	if s.Deaths == 0 {
		return float64(s.Kills + s.Assists)
	}
	return float64(s.Kills+s.Assists) / float64(s.Deaths)
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

// Team represents a team of CS2 players.
type Team [5]Player

// String name returns the team members separated by commas.
func (t Team) String() string {
	names := make([]string, 5)
	for i, p := range t {
		names[i] = p.DiscordID.String()
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
