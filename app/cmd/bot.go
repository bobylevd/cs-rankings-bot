package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sync/errgroup"

	"github.com/bobylevd/cs-rankings-bot/app/event"
	"github.com/bobylevd/cs-rankings-bot/app/store"
)

// Bot is a command to run discord bot.
type Bot struct {
	Token          string   `long:"token"            env:"TOKEN"            description:"Discord bot token"`
	AdminIDs       []string `long:"admin-id"         env:"ADMIN_IDS"        description:"Admin discords IDs"`
	VoiceChannelID string   `long:"voice-channel-id" env:"VOICE_CHANNEL_ID" description:"Common voice channel ID"`
	StoreLocation  string   `long:"loc"              env:"LOCATION"         description:"Store location"`
}

// Execute runs the command.
func (b Bot) Execute([]string) error {
	s, err := store.New(b.StoreLocation)
	if err != nil {
		return fmt.Errorf("init store: %w", err)
	}

	disc := &event.Discord{
		Token:          b.Token,
		AdminIDs:       b.AdminIDs,
		VoiceChannelID: b.VoiceChannelID,
		Service:        &store.Service{Store: s},
	}

	ctx, cancel := context.WithCancelCause(context.Background())
	go func() { // catch signal and invoke graceful termination
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
		sig := <-stop
		log.Printf("[WARN] caught signal: %s", sig)
		cancel(fmt.Errorf("caught signal: %s", sig))
	}()

	ewg, ctx := errgroup.WithContext(ctx)
	ewg.Go(func() error {
		log.Printf("[INFO] starting bot")
		return disc.Run(ctx)
	})
	ewg.Go(func() error {
		<-ctx.Done()
		log.Printf("[INFO] stopping bot")
		return nil
	})

	if err := ewg.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	return nil
}
