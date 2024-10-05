package event

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bobylevd/cs-rankings-bot/app/store"
	"github.com/bwmarrin/discordgo"
	"github.com/syohex/go-texttable"
	"golang.org/x/sync/errgroup"
)

// Discord is a handler for Discord commands.
type Discord struct {
	Token          string
	AdminIDs       []string
	VoiceChannelID string
	Service        *store.Service
	HandlerTimeout time.Duration
	se             *discordgo.Session
}

// Run runs the Discord handler.
// Blocking call.
func (d *Discord) Run(ctx context.Context) error {
	if d.HandlerTimeout == 0 {
		d.HandlerTimeout = 5 * time.Second
	}

	se, err := discordgo.New(fmt.Sprintf("Bot %s", d.Token))
	if err != nil {
		return fmt.Errorf("open discord session: %w", err)
	}

	d.se = se

	log.Printf("[INFO] opening discord session")
	if err := d.se.Open(); err != nil {
		return fmt.Errorf("open discord session: %w", err)
	}

	d.se.Identify.Intents = discordgo.IntentsGuildMessages
	d.se.AddHandler(d.onMessage)

	<-ctx.Done()

	log.Printf("[WARN] stopping bot with reason: %v", context.Cause(ctx))
	if err := d.se.Close(); err != nil {
		return fmt.Errorf("close discord session: %w", err)
	}

	return nil
}

func (d *Discord) onMessage(s *discordgo.Session, msg *discordgo.MessageCreate) {
	if msg.Author.ID == s.State.User.ID {
		return // ignore messages from the bot
	}

	log.Printf("[DEBUG] received message from %s: %s", msg.ChannelID, msg.Content)

	msg.Content = strings.TrimSpace(msg.Content)
	if msg.Content == "" || !strings.HasPrefix(msg.Content, "!") {
		return // do nothing
	}

	ctx, cancel := context.WithTimeout(context.Background(), d.HandlerTimeout)
	defer cancel()

	ctx = context.WithValue(ctx, senderIDKey{}, msg.Author.ID)

	var command func(ctx context.Context, args []string) (reply string, err error)
	args := strings.Fields(msg.Content)[1:] // first word is the command itself

	switch content := strings.TrimSpace(msg.Content); {
	case strings.HasPrefix(content, "!selectteams") && d.isAdmin(msg.Author.ID):
		command = d.selectTeams
	case strings.HasPrefix(content, "!stat"):
		command = d.stat
	case strings.HasPrefix(content, "!register"):
		command = d.register
	case strings.HasPrefix(content, "!toggleCore") && d.isAdmin(msg.Author.ID):
		command = d.toggleCore
	case strings.HasPrefix(content, "!ping"):
		command = d.ping
	case strings.HasPrefix(content, "!help"):
		command = d.help
	default:
		return // do nothing
	}

	replyTo := &discordgo.MessageReference{MessageID: msg.ID, ChannelID: msg.ChannelID}
	reply, err := command(ctx, args)
	if err != nil {
		log.Printf("[WARN] failed to execute command: %v", err)
		reply = "failed to execute command, check logs"
	}
	if _, err = s.ChannelMessageSendReply(msg.ChannelID, reply, replyTo); err != nil {
		log.Printf("[WARN] failed to send message: %v", err)
	}
}

func (d *Discord) selectTeams(ctx context.Context, args []string) (string, error) {
	ch, err := d.se.Channel(d.VoiceChannelID)
	if err != nil {
		return "", fmt.Errorf("get voice channel: %w", err)
	}

	var memberDiscordIDs []string
	for _, member := range ch.Members {
		memberDiscordIDs = append(memberDiscordIDs, member.UserID)
	}

	req := store.SelectTeamsRequest{PresentDiscordIDs: memberDiscordIDs}
	omitFilled := false
	for _, arg := range args {
		if arg == "|" {
			omitFilled = true
			continue
		}

		if omitFilled {
			req.OmitPlayerNames = append(req.OmitPlayerNames, arg)
		} else {
			req.PresentDiscordIDs = append(req.PresentDiscordIDs, arg)
		}
	}

	teams, err := d.Service.SelectTeams(ctx, req)
	if err != nil {
		if errors.Is(err, store.ErrNotEnoughPlayers) {
			return "not enough players to start a match", nil
		}

		var missing store.ErrMissing
		if errors.As(err, &missing) {
			// replace discord IDs with usernames
			for idx, discordID := range missing {
				u, err := d.se.User(discordID)
				if err != nil {
					log.Printf("[WARN] failed to get user %s: %v", discordID, err)
				}

				missing[idx] = u.Username
			}

			return missing.Error(), nil
		}

		return "", fmt.Errorf("select teams: %w", err)
	}

	return teams.String(), nil
}

func (d *Discord) stat(ctx context.Context, discordIDs []string) (string, error) {
	if len(discordIDs) == 0 {
		discordIDs = []string{senderID(ctx)}
	}

	for idx := range discordIDs {
		discordIDs[idx] = d.parseDiscordRef(discordIDs[idx])
	}

	if discordIDs[0] == "all" {
		discordIDs = []string{} // means "list all"
	}

	players, err := d.Service.List(ctx, discordIDs)
	if err != nil {
		var missing store.ErrMissing
		if errors.As(err, &missing) {
			return missing.Error(), nil
		}
		return "", fmt.Errorf("list players: %w", err)
	}

	mu := &sync.Mutex{}
	tbl := &texttable.TextTable{}
	_ = tbl.SetHeader(
		"Name",
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

	ewg, ctx := errgroup.WithContext(ctx)
	for _, pl := range players {
		pl := pl
		ewg.Go(func() error {
			u, err := d.se.User(string(pl.DiscordID))
			if err != nil {
				log.Printf("[WARN] failed to get user %s: %v", pl.DiscordID, err)
				u = &discordgo.User{Username: string(pl.DiscordID)}
			}

			mu.Lock()
			defer mu.Unlock()

			_ = tbl.AddRow(
				u.Username,
				pl.SteamID,
				strconv.FormatBool(pl.CoreMember),
				fmt.Sprintf("%.2f", pl.ELO),
				strconv.Itoa(pl.GamesPlayed),
				strconv.Itoa(pl.Wins),
				fmt.Sprintf("%.2f%%", pl.WinRate()*100),
				strconv.Itoa(pl.Kills),
				strconv.Itoa(pl.Assists),
				strconv.Itoa(pl.Deaths),
				fmt.Sprintf("%.2f", pl.KDA()),
			)

			return nil
		})
	}

	if err := ewg.Wait(); err != nil {
		return "", fmt.Errorf("get usernames: %w", err)
	}

	return "```\n" + tbl.Draw() + "\n```", nil
}

func (d *Discord) register(ctx context.Context, args []string) (string, error) {
	if len(args) != 2 {
		return "usage: !register <steamid>", nil
	}

	steamID := args[0]
	if err := d.Service.Register(ctx, senderID(ctx), steamID); err != nil {
		return "", fmt.Errorf("register player: %w", err)
	}

	return "player registered", nil
}

func (d *Discord) toggleCore(ctx context.Context, args []string) (string, error) {
	if len(args) != 1 {
		return "usage: !toggleCore <name>", nil
	}

	isCore, err := d.Service.ToggleCore(ctx, args[0])
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return "player not found", nil
		}

		return "", fmt.Errorf("toggle core: %w", err)
	}

	return fmt.Sprintf("core status toggled, set: %t", isCore), nil
}

func (d *Discord) isAdmin(discordID string) bool {
	for _, id := range d.AdminIDs {
		if discordID == id {
			return true
		}
	}
	return false
}

func (d *Discord) ping(context.Context, []string) (string, error) { return "pong!", nil }

func (d *Discord) parseDiscordRef(ref string) string {
	if strings.HasPrefix(ref, "<@!") && strings.HasSuffix(ref, ">") {
		return ref[3 : len(ref)-1]
	}
	return ref
}

func (d *Discord) help(context.Context, []string) (reply string, err error) {
	return `
!selectteams [omitPlayer1 omitPlayer2 ...] | [requiredPlayer1 requiredPlayer2 ...] - запуск шафла команд
!stat [discordID1 discordID2 ... | all] - статистика игрока/игроков
!register <steamid> - регистрация по steam id
!toggleCore <name> - только для админов, переключает статус "основного" игрока
!ping - pong!
!help - это сообщение
	`, nil
}

type senderIDKey struct{}

func senderID(ctx context.Context) string {
	if v := ctx.Value(senderIDKey{}); v != nil {
		return v.(string)
	}
	return ""
}
