package http

import (
	"fmt"
	"log"
	"modwithfriends"
	"modwithfriends/bot"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

const hackyAuthHeader = "X-LMAO-OOPS"

type groupResponse struct {
	ModuleCode modwithfriends.ModuleCode `json:"module"`
	Members    int                       `json:"members"`
}

type groupsHandler struct {
	Router       *gin.Engine
	Bot          modwithfriends.Bot
	GroupService modwithfriends.GroupService
	UserService  modwithfriends.UserService
	Pwd          string
}

func (gh *groupsHandler) register() {
	v0 := gh.Router.Group("/api/v0/groups")
	v0Protected := gh.Router.Group("/api/v0/groups", gh.hackyAuth)

	v0.GET("/", gh.getGroupsBy)

	v0Protected.GET("/:groupID", gh.getGroupByID)
	v0Protected.GET("/incomplete", gh.getIncompleteGroups)
	v0Protected.PATCH("/:groupID", gh.updateGroup)
}

func (gh *groupsHandler) hackyAuth(c *gin.Context) {
	token := c.GetHeader(hackyAuthHeader)
	if token != gh.Pwd {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	c.Next()
}

func (gh *groupsHandler) getIncompleteGroups(c *gin.Context) {
	groups, err := gh.GroupService.GroupsBy(modwithfriends.GroupQuery{
		Invited: false,
		MemberCriteriaQuery: &modwithfriends.MemberCriteriaQuery{
			Condition: modwithfriends.MoreThanOrEqual,
			Count:     5,
		},
	})
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, groups)
}

func (gh *groupsHandler) getGroupByID(c *gin.Context) {
	groupID := c.Param("groupID")

	group, err := gh.GroupService.Group(groupID)
	if err == modwithfriends.ErrEntityNotFound {
		c.AbortWithStatusJSON(http.StatusNotFound, newStandardResponse("Nope, doesn't exist"))
		return
	}
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, group)
}

func (gh *groupsHandler) updateGroup(c *gin.Context) {
	groupID := c.Param("groupID")

	groupToUpdate, err := gh.GroupService.Group(groupID)
	if err == modwithfriends.ErrEntityNotFound {
		c.AbortWithStatusJSON(http.StatusNotFound, newStandardResponse("Nope, doesn't exist"))
		return
	}
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, err)
		return
	}

	if err := c.ShouldBindJSON(&groupToUpdate); err != nil || groupToUpdate.ID != groupID {
		c.AbortWithStatusJSON(http.StatusBadRequest, err)
		return
	}

	err = gh.GroupService.UpdateGroup(groupID, groupToUpdate)
	if err != nil {
		log.Println(err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, err)
		return
	}

	broadcastFailures := []modwithfriends.BroadcastFailure{}

	if groupToUpdate.InviteLink != nil {
		broadcastFailures = gh.Bot.Broadcast(
			groupToUpdate.Members,
			fmt.Sprintf("Your mod group for %s is ready at: %s", groupToUpdate.ModuleCode, *groupToUpdate.InviteLink),
			nil,
		)
	}

	deletionErrors := []string{}

	for _, failure := range broadcastFailures {
		if failure.Reason == bot.ErrUserDeactivated {
			err := gh.UserService.DeleteUser(failure.User)
			if err != nil {
				deletionErrors = append(deletionErrors, fmt.Sprintf("Failed to delete user %d: %s", failure.User, err.Error()))
			}
		}
	}

	c.JSON(http.StatusOK, broadcastResponse{
		Message:       "Yee update is successful",
		FailedToReach: broadcastFailures,
		Errors:        deletionErrors,
	})
}

func (gh *groupsHandler) getGroupsBy(c *gin.Context) {
	var memberCount *int
	var moduleCode *modwithfriends.ModuleCode

	memberCountQuery, sizeExist := c.GetQuery("size")
	if sizeExist {
		val, err := strconv.Atoi(memberCountQuery)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest,
				newStandardResponse("Please provide a valid integer for the size query"))
			return
		}
		memberCount = &val
	}

	moduleCodeQuery, moduleCodeExist := c.GetQuery("module")
	moduleCodeQueryClean := strings.ReplaceAll(strings.ToUpper(moduleCodeQuery), " ", "")
	if moduleCodeExist && moduleCodeQueryClean == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest,
			newStandardResponse("Please provide a valid module code for the module query"))
		return
	}
	if moduleCodeExist && moduleCodeQueryClean != "" {
		val := modwithfriends.ModuleCode(moduleCodeQueryClean)
		moduleCode = &val
	}

	var memberCriteriaQuery *modwithfriends.MemberCriteriaQuery
	if memberCount != nil {
		memberCriteriaQuery = &modwithfriends.MemberCriteriaQuery{
			Condition: modwithfriends.Equal,
			Count:     *memberCount,
		}
	}

	groups, err := gh.GroupService.GroupsBy(modwithfriends.GroupQuery{
		Invited:             false,
		ModuleCode:          moduleCode,
		MemberCriteriaQuery: memberCriteriaQuery,
	})
	if err != nil {
		log.Println("Critical error occurred with GroupsBy endpoint: " + err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError,
			newStandardResponse("Hmmmm, something is not right 🖕"))
		return
	}

	declassifiedGroups := []groupResponse{}
	for _, group := range groups {
		declassifiedGroups = append(declassifiedGroups, groupResponse{
			ModuleCode: group.ModuleCode,
			Members:    len(group.Members),
		})
	}

	c.JSON(http.StatusOK, declassifiedGroups)
}
