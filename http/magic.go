package http

import (
	"fmt"
	"modwithfriends"
	"modwithfriends/bot"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type broadcastRequest struct {
	Message string `json:"message"`
}

type broadcastResponse struct {
	Message       string                            `json:"message"`
	FailedToReach []modwithfriends.BroadcastFailure `json:"failedToReach"`
	Errors        []string                          `json:"errors"`
}

// TODO: Clearly the APIs are a whack job, probably needs to be re-written lmao.
type magicHandler struct {
	Router      *gin.Engine
	Bot         modwithfriends.Bot
	UserService modwithfriends.UserService
	Pwd         string
}

func (mh *magicHandler) register() {
	v0 := mh.Router.Group("/api/v0/magic", mh.hackyAuth)

	v0.POST("/broadcast", mh.handleBroadcast)
}

func (mh *magicHandler) hackyAuth(c *gin.Context) {
	token := c.GetHeader(hackyAuthHeader)
	if token != mh.Pwd {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	c.Next()
}

func (mh *magicHandler) handleBroadcast(c *gin.Context) {
	req := broadcastRequest{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, err)
		return
	}

	users, err := mh.UserService.Users()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, err)
		return
	}

	broadcastFailures := mh.Bot.Broadcast(users, req.Message, &modwithfriends.BroadcastRate{
		Rate:  20,
		Delay: 1 * time.Second,
	})

	deletionErrors := []string{}

	for _, failure := range broadcastFailures {
		if failure.Reason == bot.ErrUserDeactivated {
			err := mh.UserService.DeleteUser(failure.User)
			if err != nil {
				deletionErrors = append(deletionErrors, fmt.Sprintf("Failed to delete user %d: %s", failure.User, err.Error()))
			}
		}
	}

	c.JSON(http.StatusOK, broadcastResponse{
		Message:       "Broadcast successful",
		FailedToReach: broadcastFailures,
		Errors:        deletionErrors,
	})
}
