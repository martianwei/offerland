package models

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base32"
	"time"

	"github.com/google/uuid"
	"github.com/pascaldekloe/jwt"
	"offerland.cc/internal/funcs"
)

// Define a Token struct to hold the data for an individual token. This includes the
// plaintext and hashed versions of the token, associated user ID, expiry time and
// scope.
// Define the TokenModel type.
type TokenModel struct {
	DB *sql.DB
}

type Token struct {
	Plaintext string    `json:"token"` // The plaintext version of the token
	Hash      []byte    `json:"-"`     // The hashed version of the token
	UserID    uuid.UUID `json:"-"`     // The ID of the user the token is associated with
	Passcode  string    `json:"passcode"`
	Expiry    time.Time `json:"expiry"` // The expiry time for the token
}

type JWTToken struct {
	Token string        `json:"token"`
	TTL   time.Duration `json:"ttl"`
}

// Generate new token pair
func (m *TokenModel) NewTokenPair(userID uuid.UUID) (*JWTToken, *JWTToken, error) {
	// Generate access token
	accessTTL := funcs.LoadEnv("ACCESS_TOKEN_TTL")
	accessTTLDuration, err := time.ParseDuration(accessTTL)
	if err != nil {
		return nil, nil, err
	}
	accessTokenSecret := funcs.LoadEnv("ACCESS_TOKEN_SECRET")
	accessToken, err := NewJWT(userID, accessTTLDuration, accessTokenSecret)
	if err != nil {
		return nil, nil, err
	}

	// Generate refresh token
	refreshTokenSecret := funcs.LoadEnv("REFRESH_TOKEN_SECRET")
	refreshTTL := funcs.LoadEnv("REFRESH_TOKEN_TTL")
	refreshTTLDuration, err := time.ParseDuration(refreshTTL)
	if err != nil {
		return nil, nil, err
	}
	refreshToken, err := NewJWT(userID, refreshTTLDuration, refreshTokenSecret)
	if err != nil {
		return nil, nil, err
	}
	m.DeleteRefreshTokenByUserID(userID)
	m.InsertRefreshToken(refreshToken, userID)
	// Return the tokens.
	return accessToken, refreshToken, nil
}

func (m *TokenModel) InsertRefreshToken(refreshToken *JWTToken, userID uuid.UUID) error {
	query := `
		INSERT INTO refresh_tokens (token, user_id, created_at, expires_at)
		VALUES ($1, $2, $3, $4)
	`
	_, err := m.DB.Exec(query, refreshToken.Token, userID, time.Now(), time.Now().Add(refreshToken.TTL))
	if err != nil {
		return err
	}
	return nil
}

func (m *TokenModel) GetUserIDByRefreshToken(refreshToken string) (uuid.UUID, error) {
	query := `
		SELECT user_id FROM refresh_tokens
		WHERE token = $1
	`
	var userID uuid.UUID
	err := m.DB.QueryRow(query, refreshToken).Scan(&userID)
	if err != nil {
		return uuid.Nil, err
	}
	return userID, nil
}

func (m *TokenModel) DeleteRefreshToken(refreshToken string) error {
	query := `
		DELETE FROM refresh_tokens
		WHERE token = $1
	`
	_, err := m.DB.Exec(query, refreshToken)
	if err != nil {
		return err
	}
	return nil
}

func (m *TokenModel) DeleteRefreshTokenByUserID(userID uuid.UUID) error {
	query := `
		DELETE FROM refresh_tokens
		WHERE user_id = $1
	`
	_, err := m.DB.Exec(query, userID)
	if err != nil {
		return err
	}
	return nil
}

func generateToken(userID uuid.UUID, ttl time.Duration) (*Token, error) {
	// Create a Token instance containing the user ID, expiry, and scope information.
	// Notice that we add the provided ttl (time-to-live) duration parameter to the
	// current time to get the expiry time?
	token := &Token{
		UserID: userID,
		Expiry: time.Now().Add(ttl),
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

func NewJWT(userID uuid.UUID, ttl time.Duration, secret string) (*JWTToken, error) {
	var claims jwt.Claims
	claims.Subject = userID.String()

	expiry := time.Now().Add(ttl)
	claims.Issued = jwt.NewNumericTime(time.Now())
	claims.NotBefore = jwt.NewNumericTime(time.Now())
	claims.Expires = jwt.NewNumericTime(expiry)

	claims.Issuer = "offerland.cc"
	claims.Audiences = []string{"offerland.cc"}

	jwtBytes, err := claims.HMACSign(jwt.HS256, []byte(secret))
	if err != nil {
		return nil, err
	}

	jwtToken := &JWTToken{
		Token: string(jwtBytes),
		TTL:   ttl,
	}

	return jwtToken, nil
}

func (m TokenModel) generatePasscode() (string, error) {
	const otpChars = "1234567890"
	var length = 6

	buffer := make([]byte, length)
	_, err := rand.Read(buffer)
	if err != nil {
		return "", err
	}

	otpCharsLength := len(otpChars)
	for i := 0; i < length; i++ {
		buffer[i] = otpChars[int(buffer[i])%otpCharsLength]
	}

	return string(buffer), nil
}

func (m TokenModel) NewActivationToken(userID uuid.UUID, ttl time.Duration) (*Token, error) {
	// Generate a new passcode
	passcode, err := m.generatePasscode()
	if err != nil {
		return nil, err
	}
	// Generate a new token
	token, err := generateToken(userID, ttl)
	if err != nil {
		return nil, err
	}
	token.Passcode = passcode
	// Insert the token into the database
	err = m.InsertActivationToken(token, passcode)
	return token, err
}

func (m TokenModel) NewResetToken(userID uuid.UUID, ttl time.Duration) (*Token, error) {
	// Generate a new token
	token, err := generateToken(userID, ttl)
	if err != nil {
		return nil, err
	}
	// Insert the token into the database
	err = m.InsertResetToken(token)
	return token, err
}

func (m TokenModel) Validate(passcode string, tokenPlaintext string) (bool, error) {
	tokenHash := sha256.Sum256([]byte(tokenPlaintext))
	query := `
		SELECT passcode
		FROM activation_tokens
		WHERE activation_tokens.hash = $1
		AND activation_tokens.expiry > $2`

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
func (m TokenModel) InsertActivationToken(token *Token, passcode string) error {
	query := `
		INSERT INTO activation_tokens (hash, user_id, passcode, expiry) VALUES ($1, $2, $3, $4)`

	args := []any{token.Hash, token.UserID, passcode, token.Expiry}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := m.DB.ExecContext(ctx, query, args...)
	return err
}

func (m TokenModel) InsertResetToken(token *Token) error {
	query := `
		INSERT INTO reset_tokens (hash, user_id, expiry) VALUES ($1, $2, $3)`

	args := []any{token.Hash, token.UserID, token.Expiry}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := m.DB.ExecContext(ctx, query, args...)
	return err
}

// DeleteAllForUser() deletes all tokens for a specific user and scope.
func (m TokenModel) DeleteActivationTokensForUser(userID uuid.UUID) error {
	query := `
		DELETE FROM activation_tokens
		WHERE user_id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := m.DB.ExecContext(ctx, query, userID)
	return err
}

func (m TokenModel) DeleteResetTokensForUser(userID uuid.UUID) error {
	query := `
		DELETE FROM reset_tokens
		WHERE user_id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := m.DB.ExecContext(ctx, query, userID)
	return err
}
