package store

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/jmoiron/sqlx"
)

// ErrNotFound indicates that the entity hasn't been found in the database.
var ErrNotFound = errors.New("not found")

// Store provides methods to store/load data.
type Store struct {
	db *sqlx.DB
}

// New prepares the database.
func New(dsn string) (*Store, error) {
	db, err := sqlx.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	const schema = `
		CREATE TABLE IF NOT EXISTS players (
			discord_id TEXT PRIMARY KEY,
			steam_id TEXT NOT NULL DEFAULT '',
			core_member BOOLEAN NOT NULL DEFAULT FALSE,
			elo DOUBLE PRECISION NOT NULL DEFAULT 0,
			games_played INTEGER NOT NULL DEFAULT 0,
			wins INTEGER NOT NULL DEFAULT 0,
			kills INTEGER NOT NULL DEFAULT 0,
			assists INTEGER NOT NULL DEFAULT 0,
			deaths INTEGER NOT NULL DEFAULT 0
		);
    `

	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("create schema: %w", err)
	}

	return &Store{db: db}, nil
}

// Update updates a bunch of players in the storage, only mutable fields are updated.
func (s *Store) Update(ctx context.Context, players ...Player) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	const query = `UPDATE players SET
                   		steam_id = :steam_id,
                   		name = :name,
                   		core_member = :core_member,
						elo = :elo,
               			games_played = :games_played,
               			wins = :wins,
               			kills = :kills,
               			assists = :assists,
               			deaths = :deaths
					WHERE discord_id = :discord_id`

	for _, pl := range players {
		if _, err := tx.NamedExecContext(ctx, query, pl); err != nil {
			return fmt.Errorf("update player: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// Create inserts a new player into the storage.
func (s *Store) Create(ctx context.Context, pl Player) error {
	const query = `INSERT INTO players (discord_id, steam_id, name) VALUES (:discord_id, :steam_id, :name)`

	if _, err := s.db.NamedExecContext(ctx, query, pl); err != nil {
		return fmt.Errorf("insert player: %w", err)
	}

	return nil
}

// List returns a list of players with the given Discord IDs.
func (s *Store) List(ctx context.Context, discordIDs []string) ([]Player, error) {
	var players []Player

	var args []any
	query := `SELECT * FROM players`

	if len(discordIDs) > 0 {
		query += fmt.Sprintf(` WHERE discord_id IN (%s)`, strings.Join(
			slices.Repeat([]string{"?"}, len(discordIDs)),
			", ",
		))
		for _, id := range discordIDs {
			args = append(args, id)
		}
	}

	if err := s.db.SelectContext(ctx, &players, query, args...); err != nil {
		return nil, fmt.Errorf("select: %w", err)
	}
	return players, nil
}

// Get returns a player by the given name.
func (s *Store) Get(ctx context.Context, name string) (Player, error) {
	var pl Player
	if err := s.db.GetContext(ctx, &pl, `SELECT * FROM players WHERE name = ?`, name); err != nil {
		return Player{}, fmt.Errorf("get player: %w", err)
	}
	return pl, nil
}
