package store

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Service wraps the database store with additional methods.
type Service struct{ Store *Store }

// ErrNotEnoughPlayers is issued when the team shuffle is requested with less than 10 players.
var ErrNotEnoughPlayers = errors.New("not enough players to start a match")

// SelectTeamsRequest is a request to select two teams.
type SelectTeamsRequest struct {
	PresentDiscordIDs  []string
	RequiredDiscordIDs []string
	OmitPlayerNames    []string
}

// ErrMissing indicates that certain players were not found in the database and
// are required to be registered.
type ErrMissing []string

// Error returns the error message.
func (e ErrMissing) Error() string {
	return fmt.Sprintf("missing players are required to register: %s",
		strings.Join(e, ", "))
}

// SelectTeams selects two teams from the list of present players.
func (s *Service) SelectTeams(ctx context.Context, req SelectTeamsRequest) (Match, error) {
	if len(req.PresentDiscordIDs) < 10 {
		return Match{}, ErrNotEnoughPlayers
	}

	players, err := s.Store.List(ctx, req.PresentDiscordIDs)
	if err != nil {
		return Match{}, fmt.Errorf("list players: %w", err)
	}

	if len(players) != len(req.PresentDiscordIDs) {
		var e ErrMissing
		for _, id := range req.PresentDiscordIDs {
			if !s.containsDiscordID(players, id) {
				e = append(e, fmt.Sprintf("<@%s>", id))
			}
		}
		return Match{}, e
	}

	var candidates []Player
	for _, pl := range players {
		if s.contains(req.RequiredDiscordIDs, string(pl.DiscordID)) {
			candidates = append(candidates, pl)
		}
	}

	if len(candidates) < 10 {
		return Match{}, ErrNotEnoughPlayers
	}

	if len(candidates) == 10 { // TODO: maybe unnecessary
		return s.shuffleTeams(candidates)
	}

	// sort out core members from the shuffle
	var core []Player
	for idx, pl := range candidates {
		if !s.contains(req.RequiredDiscordIDs, string(pl.DiscordID)) && !pl.CoreMember {
			continue
		}

		core = append(core, pl)
		candidates = append(candidates[:idx], candidates[idx+1:]...)
	}

	// cut off the remaining players based on the amount of games played
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].GamesPlayed < candidates[j].GamesPlayed
	})
	candidates = candidates[:10-len(core)]

	// add core members back to the pool
	candidates = append(candidates, core...)
	return s.shuffleTeams(candidates)
}

// shuffleTeams shuffles the players among two teams fairly, based on their ELO.
func (s *Service) shuffleTeams(candidates []Player) (Match, error) {
	// sort players by ELO
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].ELO < candidates[j].ELO
	})

	// split the players into two teams
	var team1, team2 []Player
	for idx, pl := range candidates {
		if idx%2 == 0 {
			team1 = append(team1, pl)
			continue
		}
		team2 = append(team2, pl)
	}

	return Match{Team(team1), Team(team2)}, nil
}

// ToggleCore toggles the core member status of the specified player.
func (s *Service) ToggleCore(ctx context.Context, name string) (isCore bool, err error) {
	pl, err := s.Store.Get(ctx, name)
	if err != nil {
		return false, fmt.Errorf("get player: %w", err)
	}

	pl.CoreMember = !pl.CoreMember
	if err := s.Store.Update(ctx, pl); err != nil {
		return false, fmt.Errorf("update player: %w", err)
	}

	return pl.CoreMember, nil
}

// List returns a list of players with the given Discord IDs.
func (s *Service) List(ctx context.Context, discordIDs []string) ([]Player, error) {
	players, err := s.Store.List(ctx, discordIDs)
	if err != nil {
		return nil, fmt.Errorf("list players: %w", err)
	}

	if len(discordIDs) != 0 && len(players) != len(discordIDs) {
		var e ErrMissing
		for _, id := range discordIDs {
			if !s.containsDiscordID(players, id) {
				e = append(e, fmt.Sprintf("<@%s>", id))
			}
		}
		return nil, e
	}

	return players, nil
}

// contains checks whether the slice contains the specified string,
// case-insensitive.
func (s *Service) contains(strs []string, str string) bool {
	for _, s := range strs {
		if strings.EqualFold(s, str) {
			return true
		}
	}
	return false
}

// containsDiscordID checks whether the slice contains the specified Discord ID.
func (s *Service) containsDiscordID(players []Player, discordID string) bool {
	for _, pl := range players {
		if string(pl.DiscordID) == discordID {
			return true
		}
	}
	return false
}

// Register registers a new player with the specified Discord ID, Steam ID, and name.
func (s *Service) Register(ctx context.Context, discordID, steamID string) error {
	pl := Player{SteamID: steamID, DiscordID: DiscordID(discordID)}

	players, err := s.Store.List(ctx, []string{discordID})
	if err != nil {
		return fmt.Errorf("list players: %w", err)
	}

	if len(players) == 0 {
		return s.Store.Create(ctx, pl)
	}

	return s.Store.Update(ctx, pl)
}
