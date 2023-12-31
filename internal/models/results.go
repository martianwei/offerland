package models

import (
	"database/sql"
	"errors"
)

type ResultModel struct {
	DB *sql.DB
}

type Result struct {
	UserID       string `json:"user_id"`
	SchoolName   string `json:"school_name"`
	MajorName    string `json:"major_name"`
	AnnounceDate string `json:"announce_date"`
	Status       string `json:"status"`
	Others       string `json:"others"`
}

var (
	ErrDuplicateResult = errors.New("duplicate result")
)

func (m *ResultModel) Delete(userID string) error {
	query := `
		DELETE FROM user_to_results
		WHERE user_id = $1
	`
	_, err := m.DB.Exec(query, userID)
	return err
}

func (m *ResultModel) Insert(userID string, schoolName string, majorName string, announceDate string, status string, others string) error {
	// Insert only unique results
	query := `
		INSERT INTO user_to_results (user_id, school_name, major_name, announce_date, status, others)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := m.DB.Exec(query, userID, schoolName, majorName, announceDate, status, others)
	return err
}

func (m *ResultModel) Get(userID string) ([]Result, error) {
	query := `
		SELECT user_id, school_name, major_name, announce_date, status, others
		FROM user_to_results
		WHERE user_id = $1
	`

	rows, err := m.DB.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []Result{}
	for rows.Next() {
		var r Result
		err = rows.Scan(&r.UserID, &r.SchoolName, &r.MajorName, &r.AnnounceDate, &r.Status, &r.Others)
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

func (m *ResultModel) GetAll() ([]Result, error) {
	query := `
		SELECT user_id, school_name, major_name, announce_date, status, others
		FROM user_to_results
	`

	rows, err := m.DB.Query(query)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	results := []Result{}
	for rows.Next() {
		var r Result
		err = rows.Scan(&r.UserID, &r.SchoolName, &r.MajorName, &r.AnnounceDate, &r.Status, &r.Others)
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}
