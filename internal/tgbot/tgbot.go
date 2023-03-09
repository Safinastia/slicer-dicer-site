package tgbot

import (
	"context"
	"errors"
	"fmt"

	"github.com/Lawliet18/shady-business-bot/internal/message"
	"github.com/NicoNex/echotron/v3"
	"github.com/rs/zerolog"
)

type Bot struct {
	log zerolog.Logger

	token            string
	chatID           int64
	notificationChan <-chan message.Message
}

type Config struct {
	Token            string
	ChatID           int64
	NotificationChan <-chan message.Message
}

func New(log zerolog.Logger, config Config) *Bot {
	return &Bot{
		log:              log.With().Str("component", "bot").Logger(),
		token:            config.Token,
		chatID:           config.ChatID,
		notificationChan: config.NotificationChan,
	}
}

func (bot *Bot) Start(ctx context.Context) error {
	if bot.token == "" {
		return errors.New("empty token")
	}
	if bot.chatID == 0 {
		return errors.New("chatID must not be 0")
	}
	if bot.notificationChan == nil {
		return errors.New("notification channel must not be nil")
	}

	api := echotron.NewAPI(bot.token)

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg := <-bot.notificationChan:
			bot.sendMessage(api, msg)
		}
	}
}

func (bot *Bot) sendMessage(api echotron.API, msg message.Message) {
	text := fmt.Sprintf("Phone: %s\nName: %s", msg.Phone, msg.Name)
	_, err := api.SendMessage(text, bot.chatID, nil)
	if err != nil {
		bot.log.Err(err).Msg("send message")
	}
}
