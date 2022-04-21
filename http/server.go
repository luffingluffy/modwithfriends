package http

import (
	"fmt"
	"modwithfriends"

	"github.com/gin-gonic/gin"
)

// Server ...
type Server struct {
	Port         int
	Router       *gin.Engine
	Bot          modwithfriends.Bot
	UserService  modwithfriends.UserService
	GroupService modwithfriends.GroupService
	Pwd          string
}

// Start ...
func (s *Server) Start() {
	handlers := []handler{
		&groupsHandler{
			Router:       s.Router,
			Bot:          s.Bot,
			UserService:  s.UserService,
			GroupService: s.GroupService,
			Pwd:          s.Pwd,
		},
		&magicHandler{
			Router:      s.Router,
			Bot:         s.Bot,
			UserService: s.UserService,
			Pwd:         s.Pwd,
		},
	}

	for _, h := range handlers {
		h.register()
	}

	s.Router.Run(fmt.Sprintf(":%d", s.Port))
}
