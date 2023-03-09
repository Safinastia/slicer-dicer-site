package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Lawliet18/shady-business-bot/internal/message"
	"github.com/Lawliet18/shady-business-bot/internal/service"
	"github.com/Lawliet18/shady-business-bot/internal/tgbot"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

const (
	ShadyBusinessChatID = -1001820130859
)

func main() {
	_ = godotenv.Load()

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log := zerolog.New(os.Stdout).With().Timestamp().Logger()

	if err := run(log); err != nil {
		log.Fatal().Err(err).Msg("something went wrong")
		return
	}
}

func run(log zerolog.Logger) error {
	msgChan := make(chan message.Message, 1)

	ctx, cancel := context.WithCancel(context.Background())

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Info().Msg("gracefully shutting down")
		cancel()
	}()

	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		bot := tgbot.New(log, tgbot.Config{
			Token:            os.Getenv("TOKEN"),
			ChatID:           ShadyBusinessChatID,
			NotificationChan: msgChan,
		})

		err := bot.Start(ctx)
		if err != nil {
			return fmt.Errorf("start bot: %w", err)
		}

		return nil
	})

	eg.Go(func() error {
		addr := fmt.Sprintf("0.0.0.0:%s", os.Getenv("PORT"))
		svc := service.New(log, addr, msgChan)

		err := svc.Start(ctx)
		if err != nil {
			return fmt.Errorf("start service: %w", err)
		}

		return nil
	})

	if err := eg.Wait(); err != nil {
		return fmt.Errorf("errgroup wait: %w", err)
	}

	log.Info().Msg("shut down was gracefull")
	return nil
}
