package models

// import (
// 	"context"
// 	"database/sql"
// 	"time"

// 	"github.com/google/uuid"
// )

// type School struct {
// 	ID   uuid.UUID `json:"school_id"`
// 	Name string    `json:"school_name"`
// }

// // Create a SchoolModel struct which wraps the connection pool
// type SchoolModel struct {
// 	DB *sql.DB
// }

// func (m SchoolModel) GetAll() (*[]School, error) {
// 	query := `SELECT * FROM schools`

// 	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
// 	defer cancel()

// 	rows, err := m.DB.QueryContext(ctx, query)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	var schools []School
// 	for rows.Next() {
// 		var school School
// 		err := rows.Scan(&school.ID, &school.Name)
// 		if err != nil {
// 			return nil, err
// 		}
// 		schools = append(schools, school)
// 	}
// 	if err = rows.Err(); err != nil {
// 		return nil, err
// 	}
// 	return &schools, nil
// }
