package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"google.golang.org/api/idtoken"
	"offerland.cc/internal/funcs"
	"offerland.cc/internal/models"
	"offerland.cc/internal/password"
	"offerland.cc/internal/request"
	"offerland.cc/internal/response"
	"offerland.cc/internal/validator"
)

func (app *application) userCheck(c *gin.Context) {
	user := app.contextGetUser(c.Request)
	if user == nil {
		return
	}
	err := response.JSON(c.Writer, http.StatusOK, envelope{"user": user})
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
	}
}

func (app *application) userSignup(c *gin.Context) {
	var input struct {
		Username  string              `json:"username"`
		Email     string              `json:"email"`
		Password  string              `json:"password"`
		Validator validator.Validator `json:"-"`
	}

	err := request.DecodeJSON(c.Writer, c.Request, &input)
	if err != nil {
		app.badRequest(c.Writer, c.Request, err)
		return
	}

	existingEmail, err := app.models.Users.GetByEmail(input.Email)
	if err != nil && !errors.Is(err, models.ErrRecordNotFound) {
		app.serverError(c.Writer, c.Request, err)
		return
	}

	existingUsername, err := app.models.Users.GetByUsername(input.Username)
	if err != nil && !errors.Is(err, models.ErrRecordNotFound) {
		app.serverError(c.Writer, c.Request, err)
		return
	}
	input.Validator.CheckField(validator.Matches(input.Email, validator.RgxEmail), "Email", "Must be a valid email address")
	input.Validator.CheckField(existingEmail == nil, "email", "Email is already in use")

	input.Validator.CheckField(existingUsername == nil, "username", "Username is already in use")

	input.Validator.CheckField(len(input.Password) >= 8, "password", "Password is too short, must be at least 8 characters")
	input.Validator.CheckField(len(input.Password) <= 72, "password", "Password is too long, must be at most 72 characters")
	input.Validator.CheckField(validator.NotIn(input.Password, password.CommonPasswords...), "password", "Password is too common")

	if input.Validator.HasErrors() {
		app.failedValidation(c.Writer, c.Request, input.Validator)
		return
	}

	user := &models.User{
		ID:        uuid.New(),
		Username:  input.Username,
		Email:     input.Email,
		Activated: false,
	}

	passwordHash, err := password.Hash(input.Password)
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
		return
	}

	user.Password = passwordHash

	err = app.models.Users.Insert(user)
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
		return
	}

	// Add the "posts:read" permission for the new user.
	err = app.models.Permissions.AddForUser(user.ID, "posts:read")
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
		return
	}
	// After the user record has been created in the database, generate a new activation
	// token for the user.
	token, err := app.models.Tokens.NewActivationToken(user.ID, 1*24*time.Hour)

	if err != nil {
		app.serverError(c.Writer, c.Request, err)
		return
	}
	// Launch a goroutine which runs an anonymous function that sends the welcome email.
	app.background(func() {
		data := map[string]any{
			"username": user.Username,
			"passcode": token.Passcode,
		}

		err = app.mailer.Send(user.Email, data, "user_activation.tmpl")
		if err != nil {
			app.logger.Error(err)
		}
	})
	err = response.JSON(c.Writer, http.StatusCreated, envelope{"token": token.Plaintext})
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
		return
	}
}

func (app *application) userSignupWithGoogle(c *gin.Context) {
	var input struct {
		ClientID  string              `json:"client_id"`
		IDToken   string              `json:"id_token"`
		Validator validator.Validator `json:"-"`
	}

	err := request.DecodeJSON(c.Writer, c.Request, &input)
	if err != nil {
		app.badRequest(c.Writer, c.Request, err)
		return
	}

	payload, err := idtoken.Validate(context.Background(), input.IDToken, funcs.LoadEnv("GOOGLE_CLIENT_ID"))
	if err != nil {
		panic(err)
	}

	input.Validator.CheckField(payload.Claims["aud"].(string) == funcs.LoadEnv("GOOGLE_CLIENT_ID"), "Server", "ClientID is not valid")
	input.Validator.CheckField(payload.Claims["iss"].(string) == "https://accounts.google.com", "Server", "ISS is not valid")

	if input.Validator.HasErrors() {
		app.failedValidation(c.Writer, c.Request, input.Validator)
		return
	}
	var user *models.User
	user, err = app.models.Users.GetByEmail(payload.Claims["email"].(string))
	if err != nil && !errors.Is(err, models.ErrRecordNotFound) {
		app.serverError(c.Writer, c.Request, err)
		return
	}
	if user == nil {
		user = &models.User{
			ID:        uuid.New(),
			Username:  payload.Claims["name"].(string),
			Email:     payload.Claims["email"].(string),
			ISS:       "google",
			SUB:       payload.Claims["sub"].(string),
			Activated: true,
		}

		err = app.models.Users.Insert(user)
		if err != nil {
			app.serverError(c.Writer, c.Request, err)
			return
		}
		// Add the "posts:read" permission for the new user.
		err = app.models.Permissions.AddForUser(user.ID, "posts:read")
		if err != nil {
			app.serverError(c.Writer, c.Request, err)
			return
		}
	}
	// Generate newJWT
	jwtToken, err := app.models.Tokens.NewJWTToken(user.ID, 24*time.Hour)
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
		return
	}
	// Encode the token to JSON and send it in the response along with a 201 Created
	// status code.
	err = response.JSON(c.Writer, http.StatusOK, envelope{"token": jwtToken})
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
	}
}

func (app *application) userActivate(c *gin.Context) {
	user := app.userVerification(c)
	if user == nil {
		return
	}

	// Update the user's activation status.
	user.Activated = true
	// Save the updated user record in our database, checking for any edit conflicts in // the same way that we did for our movie records.
	err := app.models.Users.Update(user)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrEditConflict):
			err = app.models.Tokens.DeleteActivationTokensForUser(user.ID)
			if err != nil {
				app.serverError(c.Writer, c.Request, err)
				return
			}
			app.editConflict(c.Writer, c.Request)
		default:
			app.serverError(c.Writer, c.Request, err)
		}
		return
	}

	jwtToken, err := app.models.Tokens.NewJWTToken(user.ID, 24*time.Hour)
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
		return
	}
	// Encode the token to JSON and send it in the response along with a 201 Created
	// status code.

	err = response.JSON(c.Writer, http.StatusOK, envelope{"token": jwtToken})
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
	}
}

func (app *application) userLogin(c *gin.Context) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := request.DecodeJSON(c.Writer, c.Request, &input)
	if err != nil {
		app.badRequest(c.Writer, c.Request, err)
		return
	}

	user, err := app.models.Users.GetByEmail(input.Email)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrRecordNotFound):
			app.invalidCredentials(c.Writer, c.Request)
		default:
			app.serverError(c.Writer, c.Request, err)
		}
		return
	}

	if !user.Activated {
		app.inactiveAccount(c.Writer, c.Request)
		return
	}

	matches, err := password.Matches(input.Password, user.Password)
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
		return
	}

	if !matches {
		app.invalidCredentials(c.Writer, c.Request)
		return
	}

	// Generate newJWT
	jwtToken, err := app.models.Tokens.NewJWTToken(user.ID, 24*time.Hour)
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
		return
	}
	// Encode the token to JSON and send it in the response along with a 201 Created
	// status code.
	err = response.JSON(c.Writer, http.StatusOK, envelope{"token": jwtToken})
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
	}
}

func (app *application) userForgotPassword(c *gin.Context) {
	var input struct {
		Email     string              `json:"email"`
		Validator validator.Validator `json:"-"`
	}

	err := request.DecodeJSON(c.Writer, c.Request, &input)
	if err != nil {
		app.badRequest(c.Writer, c.Request, err)
		return
	}

	user, err := app.models.Users.GetByEmail(input.Email)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrRecordNotFound):
			app.emailNotFound(c.Writer, c.Request)
		default:
			app.serverError(c.Writer, c.Request, err)
		}
		return
	}

	if !user.Activated {
		app.inactiveAccount(c.Writer, c.Request)
		return
	}

	token, err := app.models.Tokens.NewResetToken(user.ID, 1*24*time.Hour)
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
		return
	}

	// Launch a goroutine which runs an anonymous function that sends the welcome email.
	app.background(func() {
		data := map[string]any{
			"username":  user.Username,
			"resetLink": fmt.Sprintf("http://%s%s/reset-password/%s", funcs.LoadEnv("DOMAIN"), funcs.LoadEnv("PORT"), token.Plaintext),
		}

		err = app.mailer.Send(user.Email, data, "user_forgot_password.tmpl")
		if err != nil {
			app.serverError(c.Writer, c.Request, err)
		}
	})
	err = response.JSON(c.Writer, http.StatusCreated, envelope{"message": "Email sent"})
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
		return
	}
}

func (app *application) userResetPassword(c *gin.Context) { // Parse the plaintext activation token from the request body.
	var input struct {
		TokenPlaintext string              `json:"token"`
		Password       string              `json:"password"`
		Validator      validator.Validator `json:"-"`
	}

	err := request.DecodeJSON(c.Writer, c.Request, &input)
	if err != nil {
		app.badRequest(c.Writer, c.Request, err)
	}

	// Extract the activation token from the request URL.
	tokenPlaintext := c.Param("token")
	input.TokenPlaintext = tokenPlaintext

	// Validate the plaintext token provided by the client.
	input.Validator.CheckField(input.TokenPlaintext != "", "token", "Token is required")
	input.Validator.CheckField(len(input.TokenPlaintext) == 26, "token", "must be 26 bytes long")

	input.Validator.CheckField(len(input.Password) >= 8, "password", "Password must be at least 8 characters")
	input.Validator.CheckField(len(input.Password) <= 72, "password", "Password must be at most 72 characters")
	input.Validator.CheckField(validator.NotIn(input.Password, password.CommonPasswords...), "password", "Password is too common")

	if input.Validator.HasErrors() {
		app.failedValidation(c.Writer, c.Request, input.Validator)
		return
	}

	// Retrieve the details of the user associated with the token using the
	// GetForToken() method (which we will create in a minute). If no matching record
	// is found, then we let the client know that the token they provided is not valid.
	user, err := app.models.Users.GetForResetToken(input.TokenPlaintext)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrRecordNotFound):
			app.notFound(c.Writer, c.Request)
		default:
			app.serverError(c.Writer, c.Request, err)
		}
		return
	}

	hashedPassword, err := password.Hash(input.Password)
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
		return
	}
	user.Password = hashedPassword
	err = app.models.Users.Update(user)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrEditConflict):
			err = app.models.Tokens.DeleteResetTokensForUser(user.ID)
			if err != nil {
				app.serverError(c.Writer, c.Request, err)
				return
			}
			app.editConflict(c.Writer, c.Request)
		default:
			app.serverError(c.Writer, c.Request, err)
		}
		return
	}
	err = app.models.Tokens.DeleteResetTokensForUser(user.ID)
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
		return
	}
	err = response.JSON(c.Writer, http.StatusOK, envelope{"message": "Password updated successfully"})
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
	}
}
