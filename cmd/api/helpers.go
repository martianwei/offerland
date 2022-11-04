package main

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pascaldekloe/jwt"
	"offerland.cc/internal/models"
)

type envelope map[string]any

func (app *application) generateJWTToken(userID uuid.UUID, ttl time.Duration) (*models.JWTToken, error) {
	var claims jwt.Claims
	claims.Subject = userID.String()

	expiry := time.Now().Add(24 * time.Hour)
	claims.Issued = jwt.NewNumericTime(time.Now())
	claims.NotBefore = jwt.NewNumericTime(time.Now())
	claims.Expires = jwt.NewNumericTime(expiry)

	claims.Issuer = app.config.baseURL
	claims.Audiences = []string{app.config.baseURL}

	jwtBytes, err := claims.HMACSign(jwt.HS256, []byte(app.config.jwt.secretKey))
	if err != nil {
		return nil, err
	}

	jwtToken := &models.JWTToken{
		Token:  string(jwtBytes),
		UserID: userID,
		Expiry: expiry,
	}
	app.models.Tokens.InsertJWT(jwtToken, expiry)

	return &models.JWTToken{
		Token:  string(jwtBytes),
		UserID: userID,
		Expiry: expiry,
	}, nil
}

// The background() helper accepts an arbitrary function as a parameter.
func (app *application) background(fn func()) { // Launch a background goroutine.
	app.wg.Add(1)

	go func() {
		defer app.wg.Done()
		// Recover any panic.
		defer func() {
			if err := recover(); err != nil {
				app.logger.Error(fmt.Errorf("%s", err))
			}
		}()
		// Execute the arbitrary function that we passed as the parameter.
		fn()
	}()
}
