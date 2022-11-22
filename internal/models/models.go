package models

import (
	"database/sql"
	"errors"
)

// Define a custom ErrRecordNotFound error. We'll return this from our Get() method when
// looking up a movie that doesn't exist in our database.
var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
)

type Models struct {
	Users       UserModel
	Permissions PermissionModel
	Tokens      TokenModel
	// Schools     SchoolModel
	// Majors      MajorModel
}

// For ease of use, we also add a New() method which returns a Models struct containing
// the initialized MovieModel.
func NewModels(db *sql.DB) *Models {
	return &Models{
		Users:       UserModel{DB: db},
		Permissions: PermissionModel{DB: db},
		Tokens:      TokenModel{DB: db},
		// Schools:     SchoolModel{DB: db},
		// Majors:      MajorModel{DB: db},
	}
}
