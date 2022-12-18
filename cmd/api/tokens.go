package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pascaldekloe/jwt"
	"offerland.cc/internal/response"
)

func (app *application) refreshToken(c *gin.Context) {
	// get refresh token from cookie
	refreshTokenCookie, err := c.Cookie("REFRESH_TOKEN")
	if err != nil {
		app.invalidAuthenticationToken(c.Writer, c.Request)
		return
	}

	// check if refresh token is valid and match database
	claims, err := jwt.HMACCheck([]byte(refreshTokenCookie), []byte(app.config.REFRESH_TOKEN_SECRET))
	if err != nil {
		app.invalidAuthenticationToken(c.Writer, c.Request)
		return
	}

	// Check if the JWT is still valid at this moment in time.
	// If not delete the token from the database and log out the user
	if !claims.Valid(time.Now()) {
		err := app.models.Tokens.DeleteRefreshToken(refreshTokenCookie)
		if err != nil {
			app.serverError(c.Writer, c.Request, err)
			return
		}
		// delete REFRESH_TOKEN cookie
		http.SetCookie(c.Writer, &http.Cookie{
			Name:    "REFRESH_TOKEN",
			Value:   "",
			Path:    "/",
			Domain:  "",
			MaxAge:  -1,
			Expires: time.Unix(0, 0),
		})

		app.invalidAuthenticationToken(c.Writer, c.Request)
		return
	}

	foundUserID, err := app.models.Tokens.GetUserIDByRefreshToken(refreshTokenCookie)
	if err != nil {

		app.invalidAuthenticationToken(c.Writer, c.Request)
		return
	}

	accessToken, refreshToken, err := app.models.Tokens.NewTokenPair(
		foundUserID,
		app.config.ACCESS_TOKEN_SECRET, app.config.ACCESS_TOKEN_TTL,
		app.config.REFRESH_TOKEN_SECRET, app.config.REFRESH_TOKEN_TTL,
	)

	if err != nil {
		app.serverError(c.Writer, c.Request, err)
		return
	}

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "REFRESH_TOKEN",
		Value:    refreshToken.Token,
		Path:     "/",
		Domain:   "",
		MaxAge:   int(refreshToken.TTL),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
	})

	err = response.JSON(c.Writer, http.StatusOK, envelope{"access_token": accessToken.Token})
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
	}
}
