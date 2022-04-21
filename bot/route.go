package bot

import (
	tb "gopkg.in/tucnak/telebot.v2"
)

type route struct {
	Endpoint interface{}
	Handler  func(*tb.Message)
}
