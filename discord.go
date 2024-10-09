package main

import "github.com/bwmarrin/discordgo"

type Discord struct {
	session *discordgo.Session
}

func NewDiscord(session *discordgo.Session) *Discord {
	return &Discord{session: session}
}

func (ds *Discord) GetPlayerName(playerID string) (string, error) {
	user, err := ds.session.User(playerID)
	if err != nil {
		return "", err
	}
	return user.Username, nil
}

func (ds *Discord) GetPlayersInVoiceChannel(guildID, voiceChannelID string) ([]string, error) {
	guild, err := ds.session.State.Guild(guildID)
	if err != nil {
		return nil, err
	}

	var playerIDs []string
	for _, vs := range guild.VoiceStates {
		if vs.ChannelID == voiceChannelID {
			playerIDs = append(playerIDs, vs.UserID)
		}
	}
	return playerIDs, nil
}
