package main

import (
	"database/sql"
	"fmt"
	_ "modernc.org/sqlite"
	"strings"
	"time"
)

type DB struct {
	db *sql.DB
}

func InitDB(dbName string) (*DB, error) {
	var err error
	db := &DB{}
	db.db, err = sql.Open("sqlite", dbName) // Make sure to import the correct SQLite driver
	if err != nil {
		return nil, err
	}

	// Create tables
	_, err = db.db.Exec(`
		CREATE TABLE IF NOT EXISTS players (
			PlayerID TEXT PRIMARY KEY,
			PlayerName TEXT,
			CoreMember INTEGER,
			Mmr INTEGER,
			GamesPlayed INTEGER,
			Wins INTEGER,
			Kills INTEGER,
			Assists INTEGER,
			Deaths INTEGER,
			Sniper BOOLEAN DEFAULT FALSE
		);
		CREATE TABLE IF NOT EXISTS matches (
			MatchID INTEGER PRIMARY KEY AUTOINCREMENT,
			Winner TEXT,
			Loser TEXT,
			FOREIGN KEY (Winner) REFERENCES players(PlayerID),
			FOREIGN KEY (Loser) REFERENCES players(PlayerID)
		);
		CREATE TABLE IF NOT EXISTS player_performances (
			PerformanceID INTEGER PRIMARY KEY AUTOINCREMENT,
			MatchID INTEGER,
			PlayerID TEXT,
			Kills INTEGER,
			Assists INTEGER,
			Deaths INTEGER,
			FOREIGN KEY (MatchID) REFERENCES matches(MatchID),
			FOREIGN KEY (PlayerID) REFERENCES players(PlayerID)
		);
		CREATE TABLE IF NOT EXISTS mmr_history (
			ID INTEGER PRIMARY KEY AUTOINCREMENT,
			PlayerID TEXT,
			Mmr INTEGER,
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
		return nil, fmt.Errorf("error creating tables: %v", err)
	}

	return db, nil
}

func (db *DB) Close() error {
	if db.db != nil {
		return db.db.Close()
	}
	return nil
}

// Save a player to the database
func (db *DB) SavePlayer(player *Player) error {
	query := `
        INSERT INTO players (PlayerID, PlayerName, CoreMember, Mmr, GamesPlayed, Wins, Kills, Assists, Deaths, Sniper)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT(PlayerID) DO UPDATE SET 
            PlayerName = excluded.PlayerName,
            CoreMember = excluded.CoreMember,
            Mmr = excluded.Mmr,
            GamesPlayed = excluded.GamesPlayed,
            Wins = excluded.Wins,
            Kills = excluded.Kills,
            Assists = excluded.Assists,
            Deaths = excluded.Deaths,
            Sniper = excluded.Sniper;
    `
	args := []interface{}{
		player.PlayerID, player.PlayerName, player.CoreMember, player.MMR,
		player.GamesPlayed, player.Wins, player.Kills, player.Assists,
		player.Deaths, player.Sniper,
	}

	if len(args) != 10 {
		return fmt.Errorf("expected 10 arguments, got %d", len(args))
	}

	_, err := db.db.Exec(query, args...)
	return err
}

// Save individual player performance to the database
func (db *DB) SavePlayerPerformance(matchID int, player *Player) error {
	_, err := db.db.Exec(`
		INSERT INTO player_performances (MatchID, PlayerID, Kills, Assists, Deaths)
		VALUES (?, ?, ?, ?, ?)
	`, matchID, player.PlayerID, player.Kills, player.Assists, player.Deaths)
	return err
}

// Retrieve a player from the database
func (db *DB) GetPlayer(playerID string) (*Player, error) {
	var player Player
	err := db.db.QueryRow(`
		SELECT PlayerID, PlayerName, CoreMember, Mmr, GamesPlayed, Wins, Kills, Assists, Deaths, Sniper FROM players WHERE PlayerID = ?
	`, playerID).Scan(
		&player.PlayerID, &player.PlayerName, &player.CoreMember, &player.MMR, &player.GamesPlayed, &player.Wins, &player.Kills, &player.Assists, &player.Deaths, &player.Sniper,
	)
	if err != nil {
		return nil, err
	}
	return &player, nil
}

// Save a match to the database and individual player performances
func (db *DB) SaveMatch(match *Match) (int, error) {
	winnerIds := match.Winner.GetPlayerIDs()
	loserIds := match.Loser.GetPlayerIDs()

	// Perform the database operation to save the basic match result (team IDs, winner)
	result, err := db.db.Exec(`
		INSERT INTO matches (Winner, Loser)
		VALUES (?, ?)
	`, winnerIds, loserIds)
	if err != nil {
		return 0, err
	}

	// Get the match ID for player performance association
	matchID, _ := result.LastInsertId()

	// Save individual performances for Team 1 players
	for _, player := range match.Winner.Players {
		err := db.SavePlayerPerformance(int(matchID), player)
		if err != nil {
			return 0, err
		}
	}

	// Save individual performances for Team 2 players
	for _, player := range match.Loser.Players {
		err := db.SavePlayerPerformance(int(matchID), player)
		if err != nil {
			return 0, err
		}
	}

	return int(matchID), nil
}

// Temporarily store teams in db
func (db *DB) StoreTeams(team1, team2 *Team) error {
	team1IDs := team1.GetPlayerIDs()
	team2IDs := team2.GetPlayerIDs()

	_, err := db.db.Exec(`
		INSERT INTO temp_teams (id, team1, team2, timestamp)
		VALUES (1, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET team1 = ?, team2 = ?, timestamp = CURRENT_TIMESTAMP;
	`, team1IDs, team2IDs, team1IDs, team2IDs)
	return err
}

func (db *DB) GetStoredTeams() (string, string, time.Time, error) {
	var team1IDs, team2IDs string
	var timestampStr string

	err := db.db.QueryRow("SELECT team1, team2, timestamp FROM temp_teams WHERE id = 1").Scan(&team1IDs, &team2IDs, &timestampStr)
	if err != nil {
		return "", "", time.Time{}, err
	}

	// Parse the timestamp string using time.RFC3339
	timestamp, err := time.Parse(time.RFC3339, timestampStr)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("failed to parse timestamp: %v", err)
	}

	return team1IDs, team2IDs, timestamp, nil
}

// Clear stored teams
func (db *DB) ClearStoredTeams() error {
	_, err := db.db.Exec("DELETE FROM temp_teams WHERE id = 1")
	return err
}

// Record MMR history for a player
func (db *DB) RecordMmrHistory(playerID string, mmr int, matchID int) error {
	_, err := db.db.Exec(`
        INSERT INTO mmr_history (PlayerID, Mmr, MatchID)
        VALUES (?, ?, ?)
    `, playerID, mmr, matchID)
	return err
}

func (db *DB) GetMmrHistory(playerID string) ([]int, []string, error) {
	rows, err := db.db.Query("SELECT Mmr, Timestamp FROM mmr_history WHERE PlayerID = ? ORDER BY Timestamp", playerID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var mmrs []int
	var timestamps []string

	for rows.Next() {
		var mmr int
		var timestamp string
		err = rows.Scan(&mmr, &timestamp)
		if err != nil {
			return nil, nil, err
		}
		mmrs = append(mmrs, mmr)
		timestamps = append(timestamps, timestamp)
	}

	if err = rows.Err(); err != nil {
		return nil, nil, err
	}

	return mmrs, timestamps, nil
}

func (db *DB) GetMatch(matchID int) (*Match, error) {
	// Retrieve the match from the database
	// Get Winner and Loser team player IDs
	var winnerIDsStr, loserIDsStr string
	err := db.db.QueryRow("SELECT Winner, Loser FROM matches WHERE MatchID = ?", matchID).Scan(&winnerIDsStr, &loserIDsStr)
	if err != nil {
		return nil, err
	}

	// Get players for Winner team
	winnerPlayerIDs := strings.Split(winnerIDsStr, ",")
	var winnerPlayers []*Player
	for _, playerID := range winnerPlayerIDs {
		player, err := db.GetPlayer(playerID)
		if err != nil {
			return nil, fmt.Errorf("failed to get player %s: %v", playerID, err)
		}
		winnerPlayers = append(winnerPlayers, player)
	}

	// Get players for Loser team
	loserPlayerIDs := strings.Split(loserIDsStr, ",")
	var loserPlayers []*Player
	for _, playerID := range loserPlayerIDs {
		player, err := db.GetPlayer(playerID)
		if err != nil {
			return nil, fmt.Errorf("failed to get player %s: %v", playerID, err)
		}
		loserPlayers = append(loserPlayers, player)
	}

	// Create the Match object
	match := &Match{
		MatchID: matchID,
		Winner: &Team{
			Players: winnerPlayers,
		},
		Loser: &Team{
			Players: loserPlayers,
		},
	}
	return match, nil
}

func (db *DB) HasMatches() (bool, error) {
	var count int
	err := db.db.QueryRow("SELECT COUNT(*) FROM matches").Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
