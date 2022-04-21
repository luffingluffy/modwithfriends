package bot

import (
	"fmt"
	"modwithfriends"
	"strings"
	"sync"

	tb "gopkg.in/tucnak/telebot.v2"
)

type reply func(*tb.Bot, *tb.User, *tb.Message)

type Routes struct {
	bot           *tb.Bot
	userService   modwithfriends.UserService
	moduleService modwithfriends.ModuleService
	groupService  modwithfriends.GroupService
	emailService   modwithfriends.EmailService
	feedbackEmail string
	lock          sync.RWMutex
}

func NewRoutes(
	us modwithfriends.UserService,
	ms modwithfriends.ModuleService,
	gs modwithfriends.GroupService,
	es modwithfriends.EmailService,
	feedbackEmail string,
) func(*tb.Bot) *Routes {
	return func(bot *tb.Bot) *Routes {
		return &Routes{
			bot:           bot,
			userService:   us,
			moduleService: ms,
			groupService:  gs,
			emailService:   es,
			feedbackEmail: feedbackEmail,
			lock:          sync.RWMutex{},
		}
	}
}

func (r *Routes) handleStart(msg *tb.Message) {
	chatID := modwithfriends.ChatID(msg.Chat.ID)

	err := r.userService.CreateUser(chatID)
	if err != nil && err != modwithfriends.ErrDuplicateEntityFound {
		r.bot.Send(msg.Sender, "Registration has failed, please contact admin!")
		return
	}

	r.bot.Send(msg.Sender,
		"Welcome to modwithfriends, a platform that allows you to connect with potential module mates via Telegram groups of 5 people 😜\n\n"+
			"/find GEX1007 - Register a module you're taking and be notified with a Telegram group invite when your team is fully assembled.\n\n"+
			"/groups - View every mod groups you're assigned to along with its progress.\n\n"+
			"/leave GEX1007 - Leave a mod group that has not been assigned a group invite link.\n\n"+
			"/feedback Your message - Let us know your thoughts and issues and we'll get back to you ASAP.\n\n"+
			"Enjoyed the bot? Forward https://tinyurl.com/fwens with your friends so we may group them with more awesome people!\n\n"+
			"For announcements about the bot, checkout our channel @modwithfriends 📢\n\n"+
			fmt.Sprintf("For all other enquiries, we can be reached at @typeunsafe or %s 📧", r.feedbackEmail))

	// If command is started by deeplink, direct to handleFind method.
	param := strings.ReplaceAll(strings.ToUpper(msg.Payload), " ", "")
	if param != "" {
		msg.Text = fmt.Sprintf("/find %s", param)
		r.handleFind(msg)
		return
	}
}

func (r *Routes) handleGroups(msg *tb.Message) {
	chatID := modwithfriends.ChatID(msg.Chat.ID)

	groups, err := r.userService.Groups(chatID)
	if err != nil {
		r.bot.Send(msg.Sender, "An unexpected error has occurred, please contact admin!")
		return
	}

	groupsMsg := "Your Mod Groups:\n"

	for index, group := range groups {
		availability := fmt.Sprintf("%d/5 Members", len(group.Members))
		if group.InviteLink != nil {
			availability = *group.InviteLink
		}
		groupsMsg += fmt.Sprintf("%d. %s - %s\n", index+1, string(group.ModuleCode), availability)
	}

	r.bot.Send(msg.Sender, groupsMsg)
}

func (r *Routes) handleFind(msg *tb.Message) {
	chatID := modwithfriends.ChatID(msg.Chat.ID)

	moduleCodeStr := strings.ReplaceAll(strings.ToUpper(msg.Payload), " ", "")
	if moduleCodeStr == "" {
		r.bot.Send(msg.Sender, "Please provide a valid module code. E.g. /find GEX1007")
		return
	}

	r.lock.Lock()
	defer r.lock.Unlock()

	moduleCode := modwithfriends.ModuleCode(moduleCodeStr)
	modExist, err := r.moduleService.Exist(moduleCode)
	if err != nil {
		r.bot.Send(msg.Sender, "An unexpected error has occurred, please contact admin!")
		return
	}

	if !modExist {
		err := r.moduleService.CreateModule(moduleCode)
		if err != nil {
			r.bot.Send(msg.Sender, "An unexpected error has occurred, please contact admin!")
			return
		}
	} else {
		inModuleGroup, err := r.isUserInModuleGroup(chatID, moduleCode)
		if err != nil {
			r.bot.Send(msg.Sender, "An unexpected error has occurred, please contact admin!")
			return
		}

		if inModuleGroup {
			r.bot.Send(msg.Sender, "You have already been assigned to a mod group, we will update you when the telegram group invite link is ready. In the meantime, you can use /groups to see your group allocation progress 😁")
			return
		}
	}

	groups, err := r.groupService.GroupsBy(modwithfriends.GroupQuery{
		ModuleCode: &moduleCode,
		Invited:    false,
		MemberCriteriaQuery: &modwithfriends.MemberCriteriaQuery{
			Condition: modwithfriends.LessThan,
			Count:     5,
		},
	})
	if err != nil {
		r.bot.Send(msg.Sender, "An unexpected error has occurred, please contact admin!")
		return
	}

	userIsAssigned := false

	for _, group := range groups {
		updatedGroup := group
		updatedGroup.Members = append(updatedGroup.Members, chatID)

		err := r.groupService.UpdateGroup(group.ID, updatedGroup)
		if err != nil {
			r.bot.Send(msg.Sender, "An unexpected error has occurred, please contact admin!")
			return
		}
		userIsAssigned = true

		break
	}

	if !userIsAssigned {
		_, err := r.groupService.CreateGroup(modwithfriends.Group{
			ModuleCode: moduleCode,
			Members:    []modwithfriends.ChatID{chatID},
		})
		if err != nil {
			r.bot.Send(msg.Sender, "An unexpected error has occurred, please contact admin!")
			return
		}
	}

	r.bot.Send(msg.Sender,
		fmt.Sprintf("We've assigned you to a %s mod group and will update you when the telegram group invite link is ready. "+
			"In the meantime, you can use /groups to see your group allocation progress 😁", moduleCode))
}

func (r *Routes) handleLeave(msg *tb.Message) {
	chatID := modwithfriends.ChatID(msg.Chat.ID)

	moduleCodeStr := strings.ReplaceAll(strings.ToUpper(msg.Payload), " ", "")
	if moduleCodeStr == "" {
		r.bot.Send(msg.Sender, "Please provide a valid module code. E.g. /leave GEX1007")
		return
	}
	moduleCode := modwithfriends.ModuleCode(moduleCodeStr)

	r.lock.Lock()
	defer r.lock.Unlock()

	modExist, err := r.moduleService.Exist(moduleCode)
	if err != nil {
		r.bot.Send(msg.Sender, "An unexpected error has occurred, please contact admin!")
		return
	}

	inModuleGroup, err := r.isUserInModuleGroup(chatID, moduleCode)
	if err != nil {
		r.bot.Send(msg.Sender, "An unexpected error has occurred, please contact admin!")
		return
	}

	if !modExist || !inModuleGroup {
		r.bot.Send(msg.Sender, "Hmmmm, looks like you can't leave a module you weren't assigned to in the first place 🤔")
		return
	}

	groups, err := r.userService.Groups(chatID)
	if err != nil {
		r.bot.Send(msg.Sender, "An unexpected error has occurred, please contact admin!")
		return
	}

	for _, group := range groups {
		if group.ModuleCode != moduleCode {
			continue
		}

		if group.InviteLink != nil {
			r.bot.Send(msg.Sender, "Hmmmm, you can't leave a group for which an invite link has been issued 😣")
			return
		}

		if len(group.Members) < 2 {
			err := r.groupService.DeleteGroup(group.ID)
			if err != nil {
				r.bot.Send(msg.Sender, "An unexpected error has occurred, please contact admin!")
				return
			}
			break
		}

		updatedGroup := group
		updatedGroup.Members = []modwithfriends.ChatID{}

		for _, member := range group.Members {
			if member != chatID {
				updatedGroup.Members = append(updatedGroup.Members, member)
			}
		}

		err := r.groupService.UpdateGroup(updatedGroup.ID, updatedGroup)
		if err != nil {
			r.bot.Send(msg.Sender, "An unexpected error has occurred, please contact admin!")
			return
		}
		break
	}

	r.bot.Send(msg.Sender, fmt.Sprintf("Yee haw, you're no longer assigned to any %s mod group 🤠", moduleCode))
}

func (r *Routes) handleFeedback(msg *tb.Message) {
	isEmptyFeedback := strings.ReplaceAll(strings.ToUpper(msg.Payload), " ", "") == ""
	if isEmptyFeedback {
		r.bot.Send(msg.Sender, "Please kindly enter your feedback. E.g. /feedback Hello this is my feedback!")
		return
	}

	err := r.emailService.Send(
		fmt.Sprintf("[Bot Feedback] @%s", msg.Sender.Username),
		[]string{r.feedbackEmail},
		fmt.Sprintf("ChatID: %d\n%s", msg.Chat.ID, msg.Payload),
	)
	if err != nil {
		r.bot.Send(msg.Sender, "An unexpected error has occurred, please contact admin!")
		return
	}
	r.bot.Send(msg.Sender, "Thank you for providing your feedback, we are on to it ASAP! 🤠")
}

func (r *Routes) handleNewUserJoin(msg *tb.Message) {
	for _, newUser := range msg.UsersJoined {
		r.bot.Send(msg.Chat,
			fmt.Sprintf("Hi there @%s, welcome to the group! 🥳 🎉"+
				"\nGet acquainted with the rest by introducing yourself! 😎", newUser.Username))
	}
}

func (r *Routes) isUserInModuleGroup(chatID modwithfriends.ChatID, code modwithfriends.ModuleCode) (bool, error) {
	groups, err := r.userService.Groups(chatID)
	if err != nil {
		return false, err
	}

	userIsInModuleGroup := false
	for _, group := range groups {
		if group.ModuleCode == code {
			userIsInModuleGroup = true
			break
		}
	}

	return userIsInModuleGroup, nil
}

func (r *Routes) get() []route {
	return []route{
		{
			Endpoint: "/start",
			Handler:  r.handleStart,
		},
		{
			Endpoint: "/groups",
			Handler:  r.handleGroups,
		},
		{
			Endpoint: "/find",
			Handler:  r.handleFind,
		},
		{
			Endpoint: "/leave",
			Handler:  r.handleLeave,
		},
		{
			Endpoint: "/feedback",
			Handler:  r.handleFeedback,
		},
		{
			Endpoint: tb.OnUserJoined,
			Handler:  r.handleNewUserJoin,
		},
	}
}
