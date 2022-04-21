package postgres

import (
	"errors"
	"fmt"
	"modwithfriends"

	"github.com/jmoiron/sqlx"
)

type ModuleService struct {
	DB *sqlx.DB
}

func (ms *ModuleService) Modules() ([]modwithfriends.ModuleCode, error) {
	const query = `SELECT id FROM modules`
	rows, err := ms.DB.Queryx(query)
	if err != nil {
		return nil, fmt.Errorf("Failed to query modules from database: %w", err)
	}
	defer rows.Close()

	modules := []modwithfriends.ModuleCode{}
	for rows.Next() {
		var moduleCode modwithfriends.ModuleCode

		err := rows.Scan(&moduleCode)
		if err != nil {
			return nil, fmt.Errorf("Failed to scan module from database: %w", err)
		}

		modules = append(modules, moduleCode)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("Error occurred with rows when querying for modules from database: %w", err)
	}

	return modules, nil
}

func (ms *ModuleService) Exist(code modwithfriends.ModuleCode) (bool, error) {
	moduleExists := false

	const query = `SELECT EXISTS (SELECT 1 FROM modules WHERE id=$1)`
	err := ms.DB.QueryRowx(query, &code).Scan(&moduleExists)
	if err != nil {
		return false, fmt.Errorf("Failed to check if module exists in database: %w", err)
	}

	return moduleExists, nil
}

func (ms *ModuleService) CreateModule(code modwithfriends.ModuleCode) error {
	const query = `INSERT INTO modules(id) VALUES($1)`
	_, err := ms.DB.Exec(query, code)
	if err != nil {
		return fmt.Errorf("Failed to add new module into database: %w", err)
	}
	return nil
}

func (ms *ModuleService) DeleteModule(code modwithfriends.ModuleCode) error {
	const query = `DELETE FROM modules WHERE id=$1`
	res, err := ms.DB.Exec(query, code)
	if err != nil {
		return fmt.Errorf("Failed to remove module from database: %w", err)
	}

	if rows, err := res.RowsAffected(); err != nil {
		return errors.New("Failed to get rows affected after removing module from database")
	} else if rows < 1 {
		return errors.New("Failed to remove module from database as rows affected is zero")
	}

	return nil
}
