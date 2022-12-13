package models

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

var AnonymousUser = &User{}

// Define a User struct to represent an individual user. Importantly, notice how we are
// using the json:"-" struct tag to prevent the Password and Version fields appearing in
// any output when we encode it to JSON. Also notice that the Password field uses the
// custom password type defined below.
type User struct {
	ID        uuid.UUID `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	Username  string    `json:"username"`
	Photo     string    `json:"photo"`
	Email     string    `json:"-"`
	Password  string    `json:"-"`
	ISS       string    `json:"-"`
	SUB       string    `json:"-"`
	Activated bool      `json:"activated"`
	Version   int       `json:"-"`
}

// Check if a User instance is the AnonymousUser.
func (u *User) IsAnonymous() bool {
	return u == AnonymousUser
}

// Define a custom ErrDuplicateEmail error.
var (
	ErrDuplicateEmail = errors.New("duplicate email")
)

// Create a UserModel struct which wraps the connection pool.
type UserModel struct {
	DB *sql.DB
}

// Insert a new record in the database for the user. Note that the id, created_at and
// version fields are all automatically generated by our database, so we use the
// RETURNING clause to read them into the User struct after the insert, in the same way
// that we did when creating a movie.
func (m UserModel) Insert(user *User) error {
	var query string
	var args []any
	switch {
	case user.SUB == "":
		query = `
		INSERT INTO users (user_id, username, email, password, activated)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING user_id, created_at, version`
		args = []any{user.ID, user.Username, user.Email, user.Password, user.Activated}

	case user.SUB != "":
		query = `
		INSERT INTO users (user_id, username, email, iss, sub, activated)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING user_id, created_at, version`
		args = []any{user.ID, user.Username, user.Email, user.ISS, user.SUB, user.Activated}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	// If the table already contains a record with this email address, then when we try
	// to perform the insert there will be a violation of the UNIQUE "users_email_key"
	// constraint that we set up in the previous chapter. We check for this error
	// specifically, and return custom ErrDuplicateEmail error instead.
	err := m.DB.QueryRowContext(ctx, query, args...).Scan(&user.ID, &user.CreatedAt, &user.Version)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return ErrDuplicateEmail
		default:
			return err
		}
	}
	return nil
}

func (m UserModel) Get(user_id uuid.UUID) (*User, error) {
	query := `
		SELECT user_id, created_at, username, email, COALESCE(password, ''), activated, version
		FROM users 
		WHERE user_id = $1`

	var user User

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, user_id).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.Username,
		&user.Email,
		&user.Password,
		&user.Activated,
		&user.Version,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return &user, nil
}

func (m UserModel) Delete(user_id uuid.UUID) error {
	query := `
		DELETE FROM users
		WHERE user_id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, user_id)
	if err != nil {
		return err
	}
	return nil
}

// Retrieve the User details from the database based on the user's email address.
// Because we have a UNIQUE constraint on the email column, this SQL query will only
// return one record (or none at all, in which case we return a ErrRecordNotFound error).
func (m UserModel) GetByEmail(email string) (*User, error) {
	query := `
		SELECT user_id, created_at, username, email, COALESCE(password, ''), activated, version
		FROM users
		WHERE email = $1`

	var user User
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := m.DB.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.CreatedAt, &user.Username, &user.Email, &user.Password, &user.Activated, &user.Version,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return &user, nil
}
func (m UserModel) GetByUsername(username string) (*User, error) {
	query := `
		SELECT user_id, created_at, username, email, COALESCE(password, ''), activated, version
		FROM users
		WHERE username = $1`

	var user User
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := m.DB.QueryRowContext(ctx, query, username).Scan(
		&user.ID, &user.CreatedAt, &user.Username, &user.Email, &user.Password, &user.Activated, &user.Version,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return &user, nil
}

// Update the details for a specific user. Notice that we check against the version
// field to help prevent any race conditions during the request cycle, just like we did
// when updating a movie. And we also check for a violation of the "users_email_key"
// constraint when performing the update, just like we did when inserting the user
// record originally.
func (m UserModel) Update(user *User) error {
	query := ` UPDATE users
		SET username = $1, email = $2, password = $3, activated = $4, version = version + 1
		WHERE user_id = $5 AND version = $6
		RETURNING version`

	args := []any{
		user.Username, user.Email, user.Password, user.Activated, user.ID, user.Version,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := m.DB.QueryRowContext(ctx, query, args...).Scan(&user.Version)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return ErrDuplicateEmail
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}
	return nil
}

func (m UserModel) GetForActivationToken(tokenPlaintext string) (*User, error) {
	// Calculate the SHA-256 hash of the plaintext token provided by the client.
	// Remember that this returns a byte *array* with length 32, not a slice.
	tokenHash := sha256.Sum256([]byte(tokenPlaintext))
	// Set up the SQL query.
	query := `
		SELECT users.user_id, users.created_at, users.username, users.email, COALESCE(users.password, ''), users.activated, users.version 
		FROM users
		INNER JOIN activation_tokens
		ON users.user_id = activation_tokens.user_id
		WHERE activation_tokens.hash = $1
		AND activation_tokens.expiry > $2`

	// Create a slice containing the query arguments. Notice how we use the [:] operator
	// to get a slice containing the token hash, rather than passing in the array (which
	// is not supported by the pq driver), and that we pass the current time as the
	// value to check against the token expiry.
	args := []any{tokenHash[:], time.Now()}
	var user User
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	// Execute the query, scanning the return values into a User struct. If no matching
	// record is found we return an ErrRecordNotFound error.
	err := m.DB.QueryRowContext(ctx, query, args...).Scan(
		&user.ID, &user.CreatedAt, &user.Username, &user.Email, &user.Password, &user.Activated, &user.Version,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	// Return the matching user.
	return &user, nil
}

func (m UserModel) GetForResetToken(tokenPlaintext string) (*User, error) {
	// Calculate the SHA-256 hash of the plaintext token provided by the client.
	// Remember that this returns a byte *array* with length 32, not a slice.
	tokenHash := sha256.Sum256([]byte(tokenPlaintext))
	// Set up the SQL query.
	query := `
		SELECT users.user_id, users.created_at, users.username, users.email, COALESCE(users.password, ''), users.activated, users.version 
		FROM users
		INNER JOIN reset_tokens
		ON users.user_id = reset_tokens.user_id
		WHERE reset_tokens.hash = $1
		AND reset_tokens.expiry > $2`

	// Create a slice containing the query arguments. Notice how we use the [:] operator
	// to get a slice containing the token hash, rather than passing in the array (which
	// is not supported by the pq driver), and that we pass the current time as the
	// value to check against the token expiry.
	args := []any{tokenHash[:], time.Now()}
	var user User
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	// Execute the query, scanning the return values into a User struct. If no matching
	// record is found we return an ErrRecordNotFound error.
	err := m.DB.QueryRowContext(ctx, query, args...).Scan(
		&user.ID, &user.CreatedAt, &user.Username, &user.Email, &user.Password, &user.Activated, &user.Version,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	// Return the matching user.
	return &user, nil
}
