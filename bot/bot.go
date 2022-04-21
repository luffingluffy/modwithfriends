package bot

import (
	"errors"
	"fmt"
	"modwithfriends"
	"time"

	tb "gopkg.in/tucnak/telebot.v2"
)

var (
	ErrUserDeactivated = errors.New("User has deactivated the use of bot")
)

type Bot struct {
	client *tb.Bot
	routes *Routes
}

func NewBot(token string, f func(*tb.Bot) *Routes) (*Bot, error) {
	client, err := tb.NewBot(tb.Settings{
		Token:  token,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})

	if err != nil {
		return nil, fmt.Errorf("Failed to start telegram bot: %w", err)
	}

	bot := &Bot{
		client: client,
		routes: f(client),
	}
	bot.registerRoutes(bot.routes.get()...)

	return bot, nil
}

func (b *Bot) Start() {
	b.client.Start()
}

func (b *Bot) registerRoutes(routes ...route) {
	for _, route := range routes {
		b.client.Handle(route.Endpoint, route.Handler)
	}
}

func (b *Bot) Broadcast(chatIDs []modwithfriends.ChatID, msg string, opts *modwithfriends.BroadcastRate) []modwithfriends.BroadcastFailure {
	broadcastFailures := []modwithfriends.BroadcastFailure{}

	for index, chatID := range chatIDs {
		if opts != nil && (index+1)%opts.Rate == 0 {
			time.Sleep(opts.Delay)
		}
		_, err := b.client.Send(&tb.User{ID: int(chatID)}, msg)
		if err != nil {
			tbErr, ok := err.(*tb.APIError)
			if ok && tbErr == tb.ErrBlockedByUser {
				err = ErrUserDeactivated
			}

			broadcastFailures = append(
				broadcastFailures,
				modwithfriends.BroadcastFailure{
					User:         chatID,
					Reason:       err,
					ReasonString: err.Error(),
				},
			)
		}
	}

	return broadcastFailures
}
