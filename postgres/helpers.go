package postgres

import (
	"fmt"
	"modwithfriends"

	"github.com/jmoiron/sqlx"
)

func groupMembers(DB *sqlx.DB, groupID string) ([]modwithfriends.ChatID, error) {
	const query = `SELECT user_id FROM memberships WHERE group_id=$1`
	rows, err := DB.Queryx(query, groupID)
	if err != nil {
		return nil, fmt.Errorf("Failed to get group's members from database: %w", err)
	}
	defer rows.Close()

	members := []modwithfriends.ChatID{}
	for rows.Next() {
		var memberID modwithfriends.ChatID

		err := rows.Scan(&memberID)
		if err != nil {
			return nil, fmt.Errorf("Failed to scan group's members from database: %w", err)
		}

		members = append(members, memberID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("Error occurred with rows when querying for group's members from database: %w", err)
	}

	return members, nil
}
