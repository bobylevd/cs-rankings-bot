package main

import (
	"database/sql"
	"errors"
	"time"
)

type TeamStorage struct {
	db                 *DB
	expirationDuration time.Duration // Default 48 hours, configurable
}

func NewTeamStorage(db *DB, expirationDuration time.Duration) *TeamStorage {
	return &TeamStorage{
		db:                 db,
		expirationDuration: expirationDuration,
	}
}

// Store teams temporarily in the database
func (ts *TeamStorage) StoreTeams(team1, team2 *Team) error {
	return ts.db.StoreTeams(team1, team2)
}

// Retrieve stored teams from the database with expiration logic
func (ts *TeamStorage) GetStoredTeams() (*Team, *Team, error) {
	team1IDs, team2IDs, timestamp, err := ts.db.GetStoredTeams()
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, errors.New("no stored teams found")
		}
		return nil, nil, err
	}

	// Check expiration
	if time.Since(timestamp) > ts.expirationDuration {
		err := ts.db.ClearStoredTeams()
		if err != nil {
			return nil, nil, errors.New("failed to clear expired teams")
		}
		return nil, nil, errors.New("stored teams have expired")
	}

	// Convert player IDs into player objects
	team1Players, err := getPlayersFromIDs(team1IDs, ts.db)
	if err != nil {
		return nil, nil, err
	}
	team2Players, err := getPlayersFromIDs(team2IDs, ts.db)
	if err != nil {
		return nil, nil, err
	}

	team1 := &Team{Name: "Team 1", Players: team1Players}
	team2 := &Team{Name: "Team 2", Players: team2Players}

	return team1, team2, nil
}

// Clear stored teams
func (ts *TeamStorage) ClearStoredTeams() error {
	return ts.db.ClearStoredTeams()
}
