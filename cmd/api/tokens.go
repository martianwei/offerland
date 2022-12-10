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
		app.logger.Warning("refresh token not found")
		app.invalidAuthenticationToken(c.Writer, c.Request)
		return
	}

	// check if refresh token is valid and match database
	claims, err := jwt.HMACCheck([]byte(refreshTokenCookie), []byte(app.config.jwt.refreshTokenSecret))
	if err != nil {
		app.logger.Warning("invalid token check", err)
		app.invalidAuthenticationToken(c.Writer, c.Request)
		return
	}

	// Check if the JWT is still valid at this moment in time.
	// If not delete the token from the database and log out the user
	if !claims.Valid(time.Now()) {
		app.logger.Warning("invalid refresh token")
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
		app.logger.Warning("invalid refresh token", err)
		app.invalidAuthenticationToken(c.Writer, c.Request)
		return
	}

	app.logger.Info("Generate token pair")
	accessToken, refreshToken, err := app.models.Tokens.NewTokenPair(foundUserID)
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
		return
	}

	app.logger.Info("Set REFRESH_TOKEN cookie", refreshToken.Token)
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
	app.logger.Info("JSON response", envelope{"access_token": accessToken.Token})
	err = response.JSON(c.Writer, http.StatusOK, envelope{"access_token": accessToken.Token})
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
	}
}
