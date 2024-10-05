package main

import (
	"database/sql"
	"errors"
	"github.com/bwmarrin/discordgo"
	"log"
	"math/rand"
	"time"
)

type Player struct {
	PlayerID    string
	PlayerName  string
	CoreMember  bool
	ELO         int
	GamesPlayed int
	Wins        int
	Kills       int
	Assists     int
	Deaths      int
}

// Retrieve or create a player
func getOrCreatePlayer(playerID string, s *discordgo.Session) *Player {
	var player Player
	err := db.QueryRow("SELECT PlayerID, PlayerName, CoreMember, ELO, GamesPlayed, Wins, Kills, Assists, Deaths FROM players WHERE PlayerID = ?", playerID).Scan(
		&player.PlayerID, &player.PlayerName, &player.CoreMember, &player.ELO, &player.GamesPlayed, &player.Wins, &player.Kills, &player.Assists, &player.Deaths,
	)

	if err == sql.ErrNoRows {
		// Player not found, fetch from Discord and create a new player
		playerName := getPlayerNameFromDiscord(s, playerID)
		player = Player{
			PlayerID:   playerID,
			PlayerName: playerName,
			ELO:        1000, // Starting ELO for new players
		}
		savePlayerStats(&player)
	} else if err != nil {
		log.Fatal("Error retrieving player from database:", err)
	}

	return &player
}

func selectPlayersForGame(s *discordgo.Session, guildID string, voiceChannelID string, takeAll bool, commentatorID string) ([]*Player, error) {
	var corePlayers []*Player
	var nonCorePlayers []*Player
	var allPlayers []*Player

	// Get the list of users in the voice channel
	voiceChannelMembers, err := getUsersInVoiceChannel(s, guildID, voiceChannelID)
	if err != nil {
		return nil, err
	}

	if len(voiceChannelMembers) == 0 {
		return nil, errors.New("no players in the voice channel")
	}

	// Process each member in the voice channel, check if they exist in the database or create them
	for _, member := range voiceChannelMembers {
		// Exclude the commentator from the selection
		if member.User.ID == commentatorID {
			continue
		}

		player := getOrCreatePlayer(member.User.ID, s)
		allPlayers = append(allPlayers, player)

		// Categorize based on CoreMember status
		if player.CoreMember {
			corePlayers = append(corePlayers, player)
		} else {
			nonCorePlayers = append(nonCorePlayers, player)
		}
	}

	// Scenario 1: Not taking all players, limit to 10
	if !takeAll && len(allPlayers) > 10 {
		finalSelection := selectTopPlayers(corePlayers, nonCorePlayers)

		team1, team2 := splitTeamsByELO(finalSelection)
		return append(team1, team2...), nil
	}

	// Scenario 2: Take all players, balance them by ELO
	team1, team2 := splitTeamsByELO(allPlayers)
	return append(team1, team2...), nil
}

// Helper function to get users currently in a voice channel
func getUsersInVoiceChannel(s *discordgo.Session, guildID, voiceChannelID string) ([]*discordgo.Member, error) {
	guild, err := s.State.Guild(guildID)
	if err != nil {
		return nil, err
	}

	var membersInVoice []*discordgo.Member
	for _, vs := range guild.VoiceStates {
		if vs.ChannelID == voiceChannelID {
			member, err := s.State.Member(guildID, vs.UserID)
			if err == nil {
				membersInVoice = append(membersInVoice, member)
			}
		}
	}
	return membersInVoice, nil
}

// Save or update player stats
func savePlayerStats(player *Player) {
	_, err := db.Exec(`
		INSERT INTO players (PlayerID, PlayerName, CoreMember, ELO, GamesPlayed, Wins, Kills, Assists, Deaths)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(PlayerID) DO UPDATE SET 
			PlayerName=excluded.PlayerName,
			CoreMember=excluded.CoreMember,
			ELO=excluded.ELO,
			GamesPlayed=excluded.GamesPlayed,
			Wins=excluded.Wins,
			Kills=excluded.Kills,
			Assists=excluded.Assists,
			Deaths=excluded.Deaths;
	`, player.PlayerID, player.PlayerName, player.CoreMember, player.ELO, player.GamesPlayed, player.Wins, player.Kills, player.Assists, player.Deaths)
	if err != nil {
		log.Fatal("Error saving player data:", err)
	}
}

func getPlayerStats(playerID string) (*Player, error) {
	var player Player
	err := db.QueryRow("SELECT PlayerID, PlayerName, CoreMember, ELO, GamesPlayed, Wins, Kills, Assists, Deaths FROM players WHERE PlayerID = ?", playerID).Scan(
		&player.PlayerID, &player.PlayerName, &player.CoreMember, &player.ELO, &player.GamesPlayed, &player.Wins, &player.Kills, &player.Assists, &player.Deaths,
	)
	if err != nil {
		return nil, err
	}
	return &player, nil
}

func selectTopPlayers(corePlayers, nonCorePlayers []*Player) []*Player {
	// Shuffle the non-core players to make the selection random
	rand.Seed(time.Now().UnixNano()) // Seed the random number generator
	rand.Shuffle(len(nonCorePlayers), func(i, j int) {
		nonCorePlayers[i], nonCorePlayers[j] = nonCorePlayers[j], nonCorePlayers[i]
	})
	finalSelection := append(corePlayers, nonCorePlayers...)

	// Limit the selection to 10 players if necessary
	if len(finalSelection) > 10 {
		finalSelection = finalSelection[:10]
	}

	return finalSelection
}
