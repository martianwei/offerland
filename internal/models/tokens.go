package models

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base32"
	"time"

	"github.com/google/uuid"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// Define a Token struct to hold the data for an individual token. This includes the
// plaintext and hashed versions of the token, associated user ID, expiry time and
// scope.
type Token struct {
	Plaintext string    `json:"token"` // The plaintext version of the token
	Hash      []byte    `json:"-"`     // The hashed version of the token
	UserID    uuid.UUID `json:"-"`     // The ID of the user the token is associated with
	Passcode  string    `json:"passcode"`
	Expiry    time.Time `json:"expiry"` // The expiry time for the token
}

type JWTToken struct {
	Token  string    `json:"token"`  // The plaintext version of the token
	UserID uuid.UUID `json:"-"`      // The ID of the user the token is associated with
	Expiry time.Time `json:"expiry"` // The expiry time for the token
}

func generateToken(userID uuid.UUID, passcode string, ttl time.Duration) (*Token, error) {
	// Create a Token instance containing the user ID, expiry, and scope information.
	// Notice that we add the provided ttl (time-to-live) duration parameter to the
	// current time to get the expiry time?
	token := &Token{
		UserID:   userID,
		Passcode: passcode,
		Expiry:   time.Now().Add(ttl),
	}
	// Initialize a zero-valued byte slice with a length of 16 bytes.
	randomBytes := make([]byte, 16)
	// Use the Read() function from the crypto/rand package to fill the byte slice with
	// random bytes from your operating system's CSPRNG. This will return an error if
	// the CSPRNG fails to function correctly.
	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, err
	}
	// Encode the byte slice to a base-32-encoded string and assign it to the token
	// Plaintext field. This will be the token string that we send to the user in their
	// welcome email. They will look similar to this:
	//
	// Y3QMGX3PJ3WLRL2YRTQGQ6KRHU
	//
	// Note that by default base-32 strings may be padded at the end with the =
	// character. We don't need this padding character for the purpose of our tokens, so
	// we use the WithPadding(base32.NoPadding) method in the line below to omit them.
	token.Plaintext = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)
	// Generate a SHA-256 hash of the plaintext token string. This will be the value
	// that we store in the `hash` field of our database table. Note that the
	// sha256.Sum256() function returns an *array* of length 32, so to make it easier to
	// work with we convert it to a slice using the [:] operator before storing it.
	hash := sha256.Sum256([]byte(token.Plaintext))
	token.Hash = hash[:]
	return token, nil
}

// Define the TokenModel type.
type TokenModel struct {
	DB *sql.DB
}

// The New() method is a shortcut which creates a new Token struct and then inserts the
// data in the tokens table.
func (m TokenModel) NewToken(userID uuid.UUID, ttl time.Duration) (*Token, error) {
	passcode, err := m.generatePasscode(userID.String())
	if err != nil {
		return nil, err
	}
	token, err := generateToken(userID, passcode, ttl)
	if err != nil {
		return nil, err
	}
	err = m.Insert(token)
	return token, err
}

func (m TokenModel) generatePasscode(userID string) (string, error) {
	secret := base32.StdEncoding.EncodeToString([]byte(userID))
	passcode, err := totp.GenerateCodeCustom(secret, time.Now(), totp.ValidateOpts{
		Period:    300,
		Skew:      1,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA512,
	})
	if err != nil {
		return "", err
	}
	return passcode, nil
}

func (m TokenModel) Validate(passcode string, tokenPlaintext string) (bool, error) {
	tokenHash := sha256.Sum256([]byte(tokenPlaintext))
	query := `
		SELECT passcode
		FROM users
		INNER JOIN tokens
		ON users.user_id = tokens.user_id
		WHERE tokens.hash = $1
		AND tokens.expiry > $2`

	args := []any{tokenHash[:], time.Now()}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	row := m.DB.QueryRowContext(ctx, query, args...)
	var storedPasscode string
	err := row.Scan(&storedPasscode)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}

	return storedPasscode == passcode, nil
}

// Insert() adds the data for a specific token to the tokens table.
func (m TokenModel) Insert(token *Token) error {
	query := `
		INSERT INTO tokens (hash, user_id, passcode, expiry) VALUES ($1, $2, $3, $4)`
	args := []any{token.Hash, token.UserID, token.Passcode, token.Expiry}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := m.DB.ExecContext(ctx, query, args...)
	return err
}

func (m TokenModel) InsertJWT(jwtToken *JWTToken, expiry time.Time) error {
	query := `
	INSERT INTO jwt_tokens (token, user_id, expiry) VALUES ($1, $2, $3)`
	args := []any{jwtToken.Token, jwtToken.UserID, jwtToken.Expiry}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := m.DB.ExecContext(ctx, query, args...)
	return err
}

func (m TokenModel) DeleteJWT(token string) error {
	query := `
	DELETE FROM jwt_tokens WHERE token = $1`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := m.DB.ExecContext(ctx, query, token)
	return err
}

// DeleteAllForUser() deletes all tokens for a specific user and scope.
func (m TokenModel) DeleteAllForUser(userID uuid.UUID) error {
	query := `
		DELETE FROM tokens
		WHERE user_id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := m.DB.ExecContext(ctx, query, userID)
	return err
}
