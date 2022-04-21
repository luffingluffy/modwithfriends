package postgres

import (
	"database/sql"
	"errors"
	"fmt"
	"modwithfriends"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type GroupService struct {
	DB *sqlx.DB
}

func (gs *GroupService) Groups() ([]modwithfriends.Group, error) {
	const query = `SELECT * FROM GROUPS`
	return gs.queryGroups(query)
}

func (gs *GroupService) Group(groupID string) (modwithfriends.Group, error) {
	group := modwithfriends.Group{}

	const query = `SELECT * FROM groups WHERE id=$1`
	err := gs.DB.QueryRowx(query, groupID).StructScan(&group)
	if err == sql.ErrNoRows {
		return modwithfriends.Group{}, modwithfriends.ErrEntityNotFound
	} else if err != nil {
		return modwithfriends.Group{}, fmt.Errorf("Failed to query group by groupID from database: %w", err)
	}

	members, err := groupMembers(gs.DB, group.ID)
	if err != nil {
		return modwithfriends.Group{}, fmt.Errorf("Failed to get group's members from database: %w", err)
	}
	group.Members = members

	return group, nil
}

func (gs *GroupService) GroupsBy(query modwithfriends.GroupQuery) ([]modwithfriends.Group, error) {
	queryArgs := []interface{}{}

	isInvitedQuery := `IS NOT NULL`
	if !query.Invited {
		isInvitedQuery = `IS NULL`
	}

	if query.MemberCriteriaQuery == nil {
		baseQuery := fmt.Sprintf(`SELECT * FROM groups WHERE invite_link %s`, isInvitedQuery)

		if query.ModuleCode != nil {
			baseQuery += ` AND module_id=$1`
			queryArgs = append(queryArgs, query.ModuleCode)
		}

		return gs.queryGroups(baseQuery, queryArgs...)
	}

	baseQuery := fmt.Sprintf(`SELECT groups.* FROM groups LEFT JOIN memberships as m ON groups.id=m.group_id WHERE invite_link %s`, isInvitedQuery)

	if query.ModuleCode != nil {
		baseQuery += ` AND module_id=$1`
		queryArgs = append(queryArgs, query.ModuleCode)
	}

	memberCriteriaQuery, err := query.MemberCriteriaQuery.String()
	if err != nil {
		return nil, fmt.Errorf("Failed to generate query for groups by member criteria: %w", err)
	}

	baseQuery += fmt.Sprintf(` GROUP BY groups.id HAVING COUNT(m.group_id) %s`, memberCriteriaQuery)

	return gs.queryGroups(baseQuery, queryArgs...)
}

func (gs *GroupService) CreateGroup(g modwithfriends.Group) (string, error) {
	tx, err := gs.DB.Beginx()
	if err != nil {
		return "", fmt.Errorf("Failed to start transaction to add new group to database: %w", err)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	groupID := uuid.New().String()
	g.ID = groupID

	const createGroupQuery = `INSERT INTO groups(id, invite_link, module_id) VALUES(:id, :invite_link, :module_id)`
	_, err = tx.NamedExec(createGroupQuery, &g)
	if err != nil {
		tx.Rollback()
		return "", fmt.Errorf("Failed to add new group into database: %w", err)
	}

	const createMemberQuery = `INSERT INTO memberships(group_id, user_id) VALUES($1, $2)`
	for _, member := range g.Members {
		_, err := tx.Exec(createMemberQuery, &groupID, &member)
		if err != nil {
			tx.Rollback()
			return "", fmt.Errorf("Failed to add members of new group into database: %w", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return "", fmt.Errorf("Failed to commit transaction to add new group into database: %w", err)
	}

	return groupID, nil
}

func (gs *GroupService) UpdateGroup(groupID string, updatedGroup modwithfriends.Group) error {
	tx, err := gs.DB.Beginx()
	if err != nil {
		return fmt.Errorf("Failed to start transaction to update group in database: %w", err)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	updatedGroup.ID = groupID

	const updateGroupQuery = `UPDATE groups SET invite_link=:invite_link, updated_at=now() WHERE id=:id`
	_, err = tx.NamedExec(updateGroupQuery, &updatedGroup)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("Failed to update group in database: %w", err)
	}

	members, err := groupMembers(gs.DB, groupID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("Failed to get group's existing members for comparison: %w", err)
	}

	existingMembers := map[modwithfriends.ChatID]modwithfriends.ChatID{}
	for _, member := range members {
		existingMembers[member] = member
	}

	membersToAdd := []modwithfriends.ChatID{}
	for _, member := range updatedGroup.Members {
		_, exist := existingMembers[member]
		if !exist {
			membersToAdd = append(membersToAdd, member)
		}
	}

	updatedMembers := map[modwithfriends.ChatID]modwithfriends.ChatID{}
	for _, member := range updatedGroup.Members {
		updatedMembers[member] = member
	}

	membersToRemove := []modwithfriends.ChatID{}
	for _, member := range members {
		_, exist := updatedMembers[member]
		if !exist {
			membersToRemove = append(membersToRemove, member)
		}
	}

	const createMemberQuery = `INSERT INTO memberships(group_id, user_id) VALUES($1, $2)`
	for _, member := range membersToAdd {
		_, err := tx.Exec(createMemberQuery, &groupID, &member)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("Failed to add new members of group into database: %w", err)
		}
	}

	const deleteMemberQuery = `DELETE FROM memberships WHERE group_id=$1 AND user_id=$2`
	for _, member := range membersToRemove {
		_, err := tx.Exec(deleteMemberQuery, &groupID, &member)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("Failed to remove members of group from database: %w", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("Failed to commit transaction to update group in database: %w", err)
	}

	return nil
}

func (gs *GroupService) DeleteGroup(groupID string) error {
	const query = `DELETE FROM groups WHERE id=$1`
	res, err := gs.DB.Exec(query, groupID)
	if err != nil {
		return fmt.Errorf("Failed to remove group from database: %w", err)
	}

	if rows, err := res.RowsAffected(); err != nil {
		return errors.New("Failed to get rows affected after removing group from database")
	} else if rows < 1 {
		return errors.New("Failed to remove group from database as rows affected is zero")
	}

	return nil
}

func (gs *GroupService) queryGroups(stmt string, args ...interface{}) ([]modwithfriends.Group, error) {
	groups := []modwithfriends.Group{}

	rows, err := gs.DB.Queryx(stmt, args...)
	if err != nil {
		return nil, fmt.Errorf("Failed to query groups from database: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		group := modwithfriends.Group{}

		err := rows.StructScan(&group)
		if err != nil {
			return nil, fmt.Errorf("Failed to scan group from database into struct: %w", err)
		}

		members, err := groupMembers(gs.DB, group.ID)
		if err != nil {
			return nil, fmt.Errorf("Failed to get groups' members from database: %w", err)
		}
		group.Members = members

		groups = append(groups, group)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("Error occurred with rows when querying for groups from database: %w", err)
	}

	return groups, nil
}
