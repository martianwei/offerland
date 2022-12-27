package main

import (
	"errors"
	"log"
	"strings"

	"github.com/gin-gonic/gin"
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
	accessToken := headerParts[1]
	token, err := app.firebaseClient.VerifyIDToken(c, accessToken)
	if err != nil {
		log.Fatalf("error verifying ID token: %v\n", err)
	}

	// Lookup the user record from the database.
	user, err := app.models.Users.Get(token.UID)
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
