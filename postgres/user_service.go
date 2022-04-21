package postgres

import (
	"errors"
	"fmt"
	"modwithfriends"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type UserService struct {
	DB *sqlx.DB
}

func (us *UserService) Users() ([]modwithfriends.ChatID, error) {
	const query = `SELECT id FROM users`
	rows, err := us.DB.Queryx(query)
	if err != nil {
		return nil, fmt.Errorf("Failed to query users from database: %w", err)
	}
	defer rows.Close()

	users := []modwithfriends.ChatID{}
	for rows.Next() {
		var user modwithfriends.ChatID

		err := rows.Scan(&user)
		if err != nil {
			return nil, fmt.Errorf("Failed to scan user from database: %w", err)
		}

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("Error occurred with rows when querying for users from database: %w", err)
	}

	return users, nil
}

func (us *UserService) CreateUser(chatID modwithfriends.ChatID) error {
	const query = `INSERT INTO users(id) VALUES($1)`
	_, err := us.DB.Exec(query, chatID)
	pqErr, ok := err.(*pq.Error)
	if ok && pqErr.Code == "23505" {
		return modwithfriends.ErrDuplicateEntityFound
	}
	if err != nil {
		return fmt.Errorf("Failed to add new user into database: %w", err)
	}
	return nil
}

func (us *UserService) Groups(chatID modwithfriends.ChatID) ([]modwithfriends.Group, error) {
	const query = `SELECT * FROM groups WHERE id IN (SELECT group_id FROM memberships WHERE user_id=$1)`
	rows, err := us.DB.Queryx(query, chatID)
	if err != nil {
		return nil, fmt.Errorf("Failed to get user's groups from database: %w", err)
	}
	defer rows.Close()

	groups := []modwithfriends.Group{}
	for rows.Next() {
		group := modwithfriends.Group{}

		err := rows.StructScan(&group)
		if err != nil {
			return nil, fmt.Errorf("Failed to scan user's group from database into struct: %w", err)
		}

		members, err := groupMembers(us.DB, group.ID)
		if err != nil {
			return nil, fmt.Errorf("Failed to get user's groups' members from database: %w", err)
		}
		group.Members = members

		groups = append(groups, group)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("Error occurred with rows when querying for user's groups from database: %w", err)
	}

	return groups, nil
}

func (us *UserService) DeleteUser(chatID modwithfriends.ChatID) error {
	const query = `DELETE FROM users WHERE id=$1`
	res, err := us.DB.Exec(query, chatID)
	if err != nil {
		return fmt.Errorf("Failed to remove user from database: %w", err)
	}

	if rows, err := res.RowsAffected(); err != nil {
		return errors.New("Failed to get rows affected after removing user from database")
	} else if rows < 1 {
		return errors.New("Failed to remove user from database as rows affected is zero")
	}

	return nil
}
