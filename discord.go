package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"strconv"
	"strings"
)

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
	return user.GlobalName, nil
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

func handleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, db *DB) {
	switch i.Type {
	case discordgo.InteractionMessageComponent:
		data := i.MessageComponentData()
		if strings.HasPrefix(data.CustomID, "report_match_stats_") {
			// Extract matchID from CustomID
			matchIDStr := strings.TrimPrefix(data.CustomID, "report_match_stats_")
			matchID, err := strconv.Atoi(matchIDStr)
			if err != nil {
				// Handle error
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Invalid match ID.",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}

			// Retrieve the match from the database
			match, err := db.GetMatch(matchID)
			if err != nil {
				// Handle error
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "Match not found.",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}

			userID := i.Member.User.ID

			// Verify that the user was part of the match
			found := false
			for _, player := range append(match.Winner.Players, match.Loser.Players...) {
				if player.PlayerID == userID {
					found = true
					break
				}
			}
			if !found {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "You were not part of this match.",
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}

			// Show modal to collect stats for the user
			showPlayerStatsModal(s, i.Interaction, matchID, userID, db)
		}
	case discordgo.InteractionModalSubmit:
		if strings.HasPrefix(i.ModalSubmitData().CustomID, "player_stats_modal_") {
			// Process the submitted stats
			handlePlayerStatsSubmission(s, i, db)
		}
	}
}

func showPlayerStatsModal(s *discordgo.Session, interaction *discordgo.Interaction, matchID int, playerID string, db *DB) {
	// Fetch player info from the database
	player, err := db.GetPlayer(playerID)
	if err != nil {
		// Handle error
		s.InteractionRespond(interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Player not found in the database.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Show modal to the user to input their own stats
	s.InteractionRespond(interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: fmt.Sprintf("player_stats_modal_%d_%s", matchID, playerID),
			Title:    fmt.Sprintf("Report Your Stats (%s)", player.PlayerName),
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "kills",
							Label:       "Kills",
							Style:       discordgo.TextInputShort,
							Placeholder: "Enter number of kills",
							Required:    true,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "assists",
							Label:       "Assists",
							Style:       discordgo.TextInputShort,
							Placeholder: "Enter number of assists",
							Required:    true,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "deaths",
							Label:       "Deaths",
							Style:       discordgo.TextInputShort,
							Placeholder: "Enter number of deaths",
							Required:    true,
						},
					},
				},
			},
		},
	})
}

func handlePlayerStatsSubmission(s *discordgo.Session, i *discordgo.InteractionCreate, db *DB) {
	// Extract matchID and playerID from CustomID
	data := i.ModalSubmitData()
	customID := data.CustomID
	parts := strings.Split(customID, "_")
	if len(parts) < 5 {
		// Invalid customID format
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Invalid interaction data.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	matchIDStr := parts[3]
	playerID := parts[4]

	matchID, err := strconv.Atoi(matchIDStr)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Invalid match ID.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Get the values from the modal
	var killsStr, assistsStr, deathsStr string
	for _, c := range data.Components {
		for _, innerC := range c.(*discordgo.ActionsRow).Components {
			input := innerC.(*discordgo.TextInput)
			switch input.CustomID {
			case "kills":
				killsStr = input.Value
			case "assists":
				assistsStr = input.Value
			case "deaths":
				deathsStr = input.Value
			}
		}
	}

	// Convert to integers
	kills, err := strconv.Atoi(killsStr)
	if err != nil {
		// Handle error
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Invalid input for kills.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	assists, err := strconv.Atoi(assistsStr)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Invalid input for assists.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	deaths, err := strconv.Atoi(deathsStr)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Invalid input for deaths.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Save the stats for the player and match
	err = db.SavePlayerPerformance(matchID, &Player{
		PlayerID: playerID,
		Kills:    kills,
		Assists:  assists,
		Deaths:   deaths,
	})
	if err != nil {
		// Handle error
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Error saving your stats.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Acknowledge the submission
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Your stats have been recorded. Thank you!",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
