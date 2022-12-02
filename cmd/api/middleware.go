package main

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pascaldekloe/jwt"
	"offerland.cc/internal/models"
)

func (app *application) authenticate(c *gin.Context) {
	token, err := c.Cookie("AUTH")
	if err != nil {
		app.contextSetUser(c, models.AnonymousUser)
		c.Next()
		return
	}

	claims, err := jwt.HMACCheck([]byte(token), []byte(app.config.jwt.secretKey))
	if err != nil {
		app.invalidAuthenticationToken(c.Writer, c.Request)
		return
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
		return
	}

	// Check that the exp claim is set and that it hasn't expired.
	if !claims.Valid(time.Now()) {
		app.models.Tokens.DeleteJWTTByUserID(userID)
		app.invalidAuthenticationToken(c.Writer, c.Request)
		return
	}

	if claims.Issuer != app.config.baseURL {
		app.invalidAuthenticationToken(c.Writer, c.Request)
		return
	}

	if !claims.AcceptAudience(app.config.baseURL) {
		app.invalidAuthenticationToken(c.Writer, c.Request)
		return
	}

	user, err := app.models.Users.Get(userID)
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
		return
	}

	if user != nil {
		app.contextSetUser(c, user)
	}

	c.Next()
}
