package main

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pascaldekloe/jwt"
	"offerland.cc/internal/models"
)

func (app *application) authenticate(c *gin.Context) {
	// Get the value of the Authorization header from the request.
	c.Set("Vary", "Authorization")
	authorizationHeader := c.Request.Header.Get("Authorization")
	if authorizationHeader == "" {
		app.contextSetUser(c, models.AnonymousUser)
		c.Next()
		return
	}

	headerParts := strings.Split(authorizationHeader, " ")
	if len(headerParts) != 2 || headerParts[0] != "Bearer" {
		app.invalidAuthenticationToken(c.Writer, c.Request)
		return
	}

	token := headerParts[1]
	claims, err := jwt.HMACCheck([]byte(token), []byte(app.config.jwt.secretKey))
	if err != nil {
		app.invalidAuthenticationToken(c.Writer, c.Request)
		return
	}

	// Check that the exp claim is set and that it hasn't expired.
	if !claims.Valid(time.Now()) {
		app.models.Tokens.DeleteJWT(token)
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

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
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
