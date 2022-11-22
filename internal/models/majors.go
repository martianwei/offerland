package models

// import (
// 	"context"
// 	"database/sql"
// 	"time"

// 	"github.com/google/uuid"
// )

// type Major struct {
// 	ID           uuid.UUID `json:"major_id"`
// 	Name         string    `json:"major_name"`
// 	SchoolID     uuid.UUID `json:"school_id"`
// 	DegreeID     string    `json:"degree_id"`
// 	DepartmentID string    `json:"department_id"`
// }

// type MajorSchoolName struct {
// 	Name       string `json:"major_name"`
// 	SchoolName string `json:"school_name"`
// 	DegreeName string `json:"degree_name"`
// }

// // Create a MajorModel struct which wraps the connection pool
// type MajorModel struct {
// 	DB *sql.DB
// }

// func (m MajorModel) GetAll() (*[]MajorSchoolName, error) {
// 	query := `SELECT majors.major_name, schools.school_name, degrees.degree_name FROM majors
// 	INNER JOIN schools ON majors.school_id = schools.school_id
// 	INNER JOIN degrees ON majors.degree_id = degrees.degree_id`

// 	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
// 	defer cancel()

// 	rows, err := m.DB.QueryContext(ctx, query)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	var majors []MajorSchoolName
// 	for rows.Next() {
// 		var major MajorSchoolName
// 		err := rows.Scan(&major.Name, &major.SchoolName, &major.DegreeName)
// 		if err != nil {
// 			return nil, err
// 		}
// 		majors = append(majors, major)
// 	}
// 	if err = rows.Err(); err != nil {
// 		return nil, err
// 	}
// 	return &majors, nil
// }

// func (m MajorModel) GetMajorsBySchool(schoolName string) (*[]MajorSchoolName, error) {
// 	query := `SELECT majors.major_name, schools.school_name, degrees.degree_name FROM majors
// 	INNER JOIN schools ON majors.school_id = schools.school_id
// 	INNER JOIN degrees ON majors.degree_id = degrees.degree_id
// 	WHERE schools.school_name = $1`

// 	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
// 	defer cancel()

// 	rows, err := m.DB.QueryContext(ctx, query, schoolName)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	var majors []MajorSchoolName
// 	for rows.Next() {
// 		var major MajorSchoolName
// 		err := rows.Scan(&major.Name, &major.SchoolName, &major.DegreeName)
// 		if err != nil {
// 			return nil, err
// 		}
// 		majors = append(majors, major)
// 	}
// 	if err = rows.Err(); err != nil {
// 		return nil, err
// 	}
// 	return &majors, nil
// }
