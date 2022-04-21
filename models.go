package modwithfriends

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrEntityNotFound       = errors.New("Entity does not exist")
	ErrDuplicateEntityFound = errors.New("Entity already exist")
)

type ChatID int
type ModuleCode string

type Model struct {
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

type Group struct {
	ID         string     `json:"groupId" db:"id"`
	ModuleCode ModuleCode `json:"moduleCode" db:"module_id"`
	InviteLink *string    `json:"inviteLink" db:"invite_link"`
	Members    []ChatID   `json:"members"`
	Model
}

type NumericComparator string

var (
	LessThan        = NumericComparator("LESS_THAN")
	LessThanOrEqual = NumericComparator("LESS_THAN_OR_EQUAL")
	Equal           = NumericComparator("EQUAL")
	MoreThanOrEqual = NumericComparator("MORE_THAN_OR_EQUAL")
	MoreThan        = NumericComparator("MORE_THAN")
)

type MemberCriteriaQuery struct {
	Condition NumericComparator
	Count     int
}

func (mcq MemberCriteriaQuery) String() (string, error) {
	const baseQuery = "%s %d"

	switch mcq.Condition {
	case LessThan:
		return fmt.Sprintf(baseQuery, "<", mcq.Count), nil
	case LessThanOrEqual:
		return fmt.Sprintf(baseQuery, "<=", mcq.Count), nil
	case Equal:
		return fmt.Sprintf(baseQuery, "=", mcq.Count), nil
	case MoreThanOrEqual:
		return fmt.Sprintf(baseQuery, ">=", mcq.Count), nil
	case MoreThan:
		return fmt.Sprintf(baseQuery, ">", mcq.Count), nil
	default:
		return "", errors.New("Member criteria query's condition is invalid")
	}
}

type GroupQuery struct {
	*ModuleCode
	*MemberCriteriaQuery
	Invited bool
}

type UserService interface {
	Users() ([]ChatID, error)
	CreateUser(chatID ChatID) error
	Groups(chatID ChatID) ([]Group, error)
	DeleteUser(chatID ChatID) error
}

type ModuleService interface {
	Modules() ([]ModuleCode, error)
	Exist(code ModuleCode) (bool, error)
	CreateModule(code ModuleCode) error
	DeleteModule(code ModuleCode) error
}

type GroupService interface {
	Groups() ([]Group, error)
	Group(groupID string) (Group, error)
	GroupsBy(query GroupQuery) ([]Group, error)
	CreateGroup(g Group) (string, error)
	UpdateGroup(groupID string, updatedGroup Group) error
	DeleteGroup(groupID string) error
}

type BroadcastFailure struct {
	User         ChatID `json:"user"`
	Reason       error  `json:"-"`
	ReasonString string `json:"reason"`
}

type BroadcastRate struct {
	Rate  int
	Delay time.Duration
}

type Bot interface {
	Start()
	Broadcast(chatIDs []ChatID, msg string, opts *BroadcastRate) []BroadcastFailure
}

type EmailService interface {
	Send(subject string, recipients []string, message string) error
}
