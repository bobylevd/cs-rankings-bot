package main

import (
	"database/sql"
	"fmt"
	"log"
	_ "modernc.org/sqlite"
)

var db *sql.DB

// Initialize the SQLite database
func initDB() {
	var err error
	db, err = sql.Open("sqlite", "match_data.db")
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS players (
			PlayerID TEXT PRIMARY KEY,
			PlayerName TEXT,
			CoreMember INTEGER,
			ELO INTEGER,
			GamesPlayed INTEGER,
			Wins INTEGER,
			Kills INTEGER,
			Assists INTEGER,
			Deaths INTEGER
		);
		CREATE TABLE IF NOT EXISTS matches (
			MatchID INTEGER PRIMARY KEY AUTOINCREMENT,
			Team1 TEXT,
			Team2 TEXT,
			WinningTeam INTEGER,
			Kills TEXT,
			Assists TEXT,
			Deaths TEXT
		);
		CREATE TABLE IF NOT EXISTS elo_history (
			ID INTEGER PRIMARY KEY AUTOINCREMENT,
			PlayerID TEXT,
			ELO INTEGER,
			MatchID INTEGER,
			Timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(PlayerID) REFERENCES players(PlayerID),
			FOREIGN KEY(MatchID) REFERENCES matches(MatchID)
		);
		CREATE TABLE IF NOT EXISTS temp_teams (
			id INTEGER PRIMARY KEY,
			team1 TEXT,
			team2 TEXT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		log.Fatal("Error creating tables:", err)
	}
}

func getPlayersFromDB() ([]string, error) {
	rows, err := db.Query("SELECT PlayerID FROM players")
	if err != nil {
		return nil, fmt.Errorf("Error fetching players from DB: %v", err)
	}
	defer rows.Close()

	var playerIDs []string
	for rows.Next() {
		var playerID string
		if err := rows.Scan(&playerID); err != nil {
			return nil, fmt.Errorf("Error scanning player ID: %v", err)
		}
		playerIDs = append(playerIDs, playerID)
	}

	if len(playerIDs) == 0 {
		return nil, fmt.Errorf("No players found in the database")
	}

	return playerIDs, nil
}
