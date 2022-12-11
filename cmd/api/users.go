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

// whoAmI returns the currently authenticated user.
func (app *application) whoAmI(c *gin.Context) {
	user := app.contextGetUser(c.Request)
	if user == models.AnonymousUser {
		return
	}

	err := response.JSON(c.Writer, http.StatusOK, envelope{"user": user})
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
	}
}

func (app *application) Signup(c *gin.Context) {
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

	// Check if the email is already in use
	user, err := app.models.Users.GetByEmail(input.Email)
	if err != nil && !errors.Is(err, models.ErrRecordNotFound) {
		app.serverError(c.Writer, c.Request, err)
		return
	}

	// If the user exists
	if user != nil && user.Activated {
		fmt.Println("user exists")
		input.Validator.AddFieldError("email", "This email address is already in use")
	}

	// If the user exists but is not activated, delete the existing user record
	if user != nil && !user.Activated {
		err = app.models.Users.Delete(user.ID)
		if err != nil {
			app.serverError(c.Writer, c.Request, err)
			return
		}
	}

	// Check if the username is already in use
	user, err = app.models.Users.GetByUsername(input.Username)
	if err != nil && !errors.Is(err, models.ErrRecordNotFound) {
		app.serverError(c.Writer, c.Request, err)
		return
	}

	// If the user exists
	if user != nil && user.Activated {
		fmt.Println("user exists username")
		input.Validator.AddFieldError("username", "This username is already in use")
	}

	// If the user exists but is not activated, delete the existing user record
	if user != nil && !user.Activated {
		err = app.models.Users.Delete(user.ID)
		if err != nil {
			app.serverError(c.Writer, c.Request, err)
			return
		}
	}

	input.Validator.CheckField(validator.Matches(input.Email, validator.RgxEmail), "email", "Must be a valid email address")
	input.Validator.CheckField(len(input.Password) >= 8, "password", "Password is too short, must be at least 8 characters")
	input.Validator.CheckField(len(input.Password) <= 72, "password", "Password is too long, must be at most 72 characters")
	input.Validator.CheckField(validator.NotIn(input.Password, password.CommonPasswords...), "password", "Password is too common")

	if input.Validator.HasErrors() {
		app.failedValidation(c.Writer, c.Request, input.Validator)
		return
	}

	// Create the new user record
	user = &models.User{
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
	activationToken, err := app.models.Tokens.NewActivationToken(user.ID, 1*24*time.Hour)

	if err != nil {
		app.serverError(c.Writer, c.Request, err)
		return
	}
	// Launch a goroutine which runs an anonymous function that sends the welcome email.
	app.background(func() {
		data := map[string]any{
			"username": user.Username,
			"passcode": activationToken.Passcode,
		}

		err = app.mailer.Send(user.Email, data, "user_activation.tmpl")
		if err != nil {
			app.logger.Error(err)
		}
	})
	err = response.JSON(c.Writer, http.StatusCreated, envelope{"activation_token": activationToken.Plaintext})
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
		return
	}
}

func (app *application) Login(c *gin.Context) {
	app.logger.Info("Login")
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
		app.logger.Warning("Invalid credentials")
		app.invalidCredentials(c.Writer, c.Request)
		return
	}

	app.logger.Info("Generate token pair")
	accessToken, refreshToken, err := app.models.Tokens.NewTokenPair(user.ID)
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
		return
	}

	app.logger.Info("Set REFRESH_TOKEN cookie", refreshToken)
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "REFRESH_TOKEN",
		Value:    refreshToken.Token,
		Path:     "/",
		Domain:   "",
		MaxAge:   int(refreshToken.TTL.Seconds()),
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

func (app *application) GoogleLogin(c *gin.Context) {
	app.logger.Info("GoogleLogin")
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
	app.logger.Info("Decode JSON", input)
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

	app.logger.Info("Generate token pair")
	accessToken, refreshToken, err := app.models.Tokens.NewTokenPair(user.ID)
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
		return
	}

	app.logger.Info("Set REFRESH_TOKEN cookie", refreshToken)
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "REFRESH_TOKEN",
		Value:    refreshToken.Token,
		Path:     "/",
		Domain:   "",
		MaxAge:   int(refreshToken.TTL.Seconds()),
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

func (app *application) Logout(c *gin.Context) {
	app.logger.Info("Logout")
	user := app.contextGetUser(c.Request)
	if user == nil {
		return
	}
	app.logger.Info("Logout user", user)

	app.logger.Info("Remove REFRESH_TOKEN cookie")
	// Remove the refresh token cookie from the user's browser.
	http.SetCookie(c.Writer, &http.Cookie{
		Name:    "REFRESH_TOKEN",
		Value:   "",
		Path:    "/",
		MaxAge:  -1,
		Expires: time.Now().Add(-time.Hour),
	})

	// Send a 204 No Content response.
	c.Status(http.StatusNoContent)
}

func (app *application) Activate(c *gin.Context) {
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

	accessToken, refreshToken, err := app.models.Tokens.NewTokenPair(user.ID)
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
			"resetLink": fmt.Sprintf("http://%s/reset-forgot-password/%s", funcs.LoadEnv("FRONTEND_URL"), token.Plaintext),
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

func (app *application) userForgotPasswordReset(c *gin.Context) { // Parse the plaintext activation token from the request body.
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
			app.invalidToken(c.Writer, c.Request)
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

func (app *application) checkEmail(c *gin.Context) {
	email := c.Param("email")
	user, err := app.models.Users.GetByEmail(email)
	// if user not activated, then email is available
	if user != nil && !user.Activated {
		err = response.JSON(c.Writer, http.StatusOK, envelope{"available": true})
		if err != nil {
			app.serverError(c.Writer, c.Request, err)
		}
		return
	}

	// if user not found, then email is available
	if err != nil {
		switch {
		case errors.Is(err, models.ErrRecordNotFound):
			err = response.JSON(c.Writer, http.StatusOK, envelope{"available": true})
			if err != nil {
				app.serverError(c.Writer, c.Request, err)
			}
		default:
			app.serverError(c.Writer, c.Request, err)
		}
		return
	}
	err = response.JSON(c.Writer, http.StatusOK, envelope{"available": false})
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
	}
}

func (app *application) checkUsername(c *gin.Context) {
	username := c.Param("username")
	user, err := app.models.Users.GetByUsername(username)
	// if user not activated, then username is available
	if user != nil && !user.Activated {
		err = response.JSON(c.Writer, http.StatusOK, envelope{"available": true})
		if err != nil {
			app.serverError(c.Writer, c.Request, err)
		}
		return
	}

	// if user not found, then username is available
	if err != nil {
		switch {
		case errors.Is(err, models.ErrRecordNotFound):
			err = response.JSON(c.Writer, http.StatusOK, envelope{"available": true})
			if err != nil {
				app.serverError(c.Writer, c.Request, err)
			}
		default:
			app.serverError(c.Writer, c.Request, err)
		}
		return
	}

	err = response.JSON(c.Writer, http.StatusOK, envelope{"available": false})
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
	}
}

func (app *application) checkAuthor(c *gin.Context) {
	user := app.contextGetUser(c.Request)
	if user == nil {
		err := response.JSON(c.Writer, http.StatusOK, envelope{"is_Author": false})
		if err != nil {
			app.serverError(c.Writer, c.Request, err)
		}
		return
	}
	authorname := c.Param("authorname")
	author, err := app.models.Users.GetByUsername(authorname)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrRecordNotFound):
			err = response.JSON(c.Writer, http.StatusOK, envelope{"is_Author": false})
			if err != nil {
				app.serverError(c.Writer, c.Request, err)
			}
		default:
			app.serverError(c.Writer, c.Request, err)
		}
		return
	}

	if author.ID == user.ID {
		err = response.JSON(c.Writer, http.StatusOK, envelope{"is_Author": true})
		if err != nil {
			app.serverError(c.Writer, c.Request, err)
		}
		return
	}

	err = response.JSON(c.Writer, http.StatusOK, envelope{"is_Author": false})
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
	}
}
