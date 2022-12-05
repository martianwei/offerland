package models

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type StatusType string

const (
	SUCCESS = "SUCCESS"
	FAIL    = "FAIL"
	WAITING = "WAITING"
)

type ApplicationResult struct {
	ID      int        `json:"id,omitempty"`
	UserID  uuid.UUID  `json:"user_id"`
	MajorID uuid.UUID  `json:"major_id"`
	Status  StatusType `json:"status"`
}

type ApplicationResultModel struct {
	DB *sql.DB
}

func (m ApplicationResultModel) Upsert(apr *ApplicationResult) error {
	var query string
	cols := []string{
		"user_id",
		"major_id",
		"status",
	}
	exCols := []string{}
	for _, col := range cols {
		exCols = append(exCols, "EXCLUDED."+col)
	}
	jsonStr, err := json.Marshal(apr)
	if err != nil {
		return err
	}
	setMap := map[string]any{}
	if err := json.Unmarshal(jsonStr, &setMap); err != nil {
		return err
	}
	index := 1
	var args []any
	var values []string
	for _, col := range cols {
		values = append(values, fmt.Sprintf("$%d", index))
		args = append(args, setMap[col])
		index++
	}
	query = fmt.Sprintf(`
		INSERT INTO application_results (%s)
		VALUES (%s)
		ON CONFLICT (user_id, major_id)
		DO UPDATE SET (%s) = (%s);
		`, strings.Join(cols, ","),
		strings.Join(values, ","),
		strings.Join(cols, ","),
		strings.Join(exCols, ","),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err = m.DB.ExecContext(ctx, query, args...)

	return err
}

func (m ApplicationResultModel) Delete(apr *ApplicationResult) error {

	var query = `
		DELETE FROM application_results 
		WHERE user_id = $1 AND major_id = $2
	`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.QueryContext(ctx, query, apr.UserID, apr.MajorID)

	if err != nil {
		return err
	}

	return nil
}

func (m ApplicationResultModel) GetApplicationResultsByUserID(userID uuid.UUID) ([]ApplicationResult, error) {
	var query string
	cols := []string{
		"user_id",
		"major_id",
		"status",
	}
	query = fmt.Sprintf(`
		SELECT %s
		FROM application_results
		WHERE user_id = $1
	`, strings.Join(cols, ","))
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := m.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var aprs []ApplicationResult
	for rows.Next() {
		var apr ApplicationResult
		err := rows.Scan(
			&apr.UserID,
			&apr.MajorID,
			&apr.Status,
		)
		if err != nil {
			return nil, err
		}
		aprs = append(aprs, apr)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return aprs, nil
}
