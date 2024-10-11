package main

import (
	"math"
)

type Player struct {
	PlayerID    string
	PlayerName  string
	CoreMember  bool
	MMR         int
	GamesPlayed int
	Wins        int
	Kills       int
	Assists     int
	Deaths      int
	Percentile  float64
	KDA         float64
	Sniper      bool
}

func (p *Player) GetPlayer(playerID string, db *DB) (*Player, error) {
	return db.GetPlayer(playerID)
}

func (p *Player) SavePlayer(player *Player, db *DB) error {
	return db.SavePlayer(player)
}

func (p *Player) CalculateKda() float64 {
	return float64(p.Kills+p.Assists) / math.Max(1.0, float64(p.Deaths))
}

func (p *Player) SaveStats(db *DB) error {
	return db.SavePlayer(p)
}
