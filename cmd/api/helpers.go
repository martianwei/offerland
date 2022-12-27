package main

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"offerland.cc/internal/models"
	"offerland.cc/internal/request"
	"offerland.cc/internal/validator"
)

type envelope map[string]any

func (app *application) pong(c *gin.Context) {
	c.String(200, "pong")
}

func (app *application) checkUsernameHelper(username string) (bool, error) {
	_, err := app.models.Users.GetByUsername(username)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (app *application) checkEmailHelper(email string) (bool, error) {
	_, err := app.models.Users.GetByEmail(email)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (app *application) userVerification(c *gin.Context) *models.User {
	// Parse the plaintext activation token from the request body.
	var input struct {
		TokenPlaintext string              `json:"token"`
		Passcode       string              `json:"passcode"`
		Validator      validator.Validator `json:"-"`
	}

	err := request.DecodeJSON(c.Writer, c.Request, &input)
	if err != nil {
		app.badRequest(c.Writer, c.Request, err)
		return nil
	}

	// Extract the activation token from the request URL.
	tokenPlaintext := c.Param("token")
	input.TokenPlaintext = tokenPlaintext
	// Validate the plaintext token provided by the client.
	input.Validator.CheckField(input.TokenPlaintext != "", "token", "Token is required")
	input.Validator.CheckField(len(input.TokenPlaintext) == 26, "token", "must be 26 bytes long")
	input.Validator.CheckField(len(input.Passcode) == 6, "token", "must be 6 bytes long")

	if input.Validator.HasErrors() {
		app.failedValidation(c.Writer, c.Request, input.Validator)
		return nil
	}

	// Retrieve the details of the user associated with the token using the
	// GetForToken() method. If no matching record
	// is found, then we let the client know that the token they provided is not valid.
	user, err := app.models.Users.GetForActivationToken(input.TokenPlaintext)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrRecordNotFound):
			app.notFound(c.Writer, c.Request)
		default:
			app.serverError(c.Writer, c.Request, err)
		}
		return nil
	}

	// Validate the passcode provided by the client.
	valid, err := app.models.Tokens.Validate(input.Passcode, input.TokenPlaintext)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrRecordNotFound):
			input.Validator.AddFieldError("passcode", "invalid or expired passcode")
			app.failedValidation(c.Writer, c.Request, input.Validator)
		default:
			app.serverError(c.Writer, c.Request, err)
		}
		return nil
	}

	if !valid {
		input.Validator.AddFieldError("passcode", "invalid or expired passcode")
		app.failedValidation(c.Writer, c.Request, input.Validator)
		return nil
	}
	// If everything went successfully, then we delete all activation tokens for the
	// user.
	err = app.models.Tokens.DeleteActivationTokensForUser(user.ID)
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
		return nil
	}
	return user
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
