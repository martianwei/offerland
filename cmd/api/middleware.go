package main

import (
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pascaldekloe/jwt"
	"offerland.cc/internal/models"
)

func (app *application) authenticate(c *gin.Context) {
	c.Header("Vary", "Authorization")
	authorizationHeader := c.GetHeader("Authorization")

	headerParts := strings.Split(authorizationHeader, " ")

	if len(headerParts) != 2 || headerParts[0] != "Bearer" {
		app.contextSetUser(c, models.AnonymousUser)
		c.Next()
		return
	}
	token := headerParts[1]

	// Parse the JWT and extract the claims. This will return an error if the JWT
	// contents doesn't match the signature (i.e. the token has been tampered with) // or the algorithm isn't valid.
	claims, err := jwt.HMACCheck([]byte(token), []byte(app.config.ACCESS_TOKEN_SECRET))
	if err != nil {
		app.invalidAuthenticationToken(c.Writer, c.Request)
		return
	}

	// Check if the JWT is still valid at this moment in time.
	if !claims.Valid(time.Now()) {
		app.expiredToken(c.Writer, c.Request)
		c.Abort()
		return
	}
	// Check that the issuer is our application.
	if claims.Issuer != "offerland.cc" {
		app.invalidAuthenticationToken(c.Writer, c.Request)
		return
	}
	// Check that our application is in the expected audiences for the JWT.
	if !claims.AcceptAudience("offerland.cc") {
		app.invalidAuthenticationToken(c.Writer, c.Request)
		return
	}
	// At this point, we know that the JWT is all OK and we can trust the data in // it. We extract the user ID from the claims subject and convert it from a // string into an int64.
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
		return
	}
	// Lookup the user record from the database.
	user, err := app.models.Users.Get(userID)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrRecordNotFound):
			app.invalidAuthenticationToken(c.Writer, c.Request)
		default:
			app.serverError(c.Writer, c.Request, err)
		}
		return
	}
	// Add the user record to the request context and continue as normal
	app.contextSetUser(c, user)
	c.Next()
}
