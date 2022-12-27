package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"
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
		input.Validator.AddFieldError("email", "This email address is already in use")
	}

	// If the user exists but is not activated, delete the existing user record
	if user != nil && !user.Activated {
		err = app.models.Users.Delete(user.ID)
		if err != nil {
			fmt.Println("Error deleting user: ", err)
			app.serverError(c.Writer, c.Request, err)
			return
		}
	}

	exists, err := app.checkUsernameHelper(input.Username)
	if err != nil && !errors.Is(err, models.ErrRecordNotFound) {
		app.serverError(c.Writer, c.Request, err)
	}

	if exists {
		fmt.Println("Username already in use: ", input.Username)
		input.Validator.AddFieldError("username", "This username is already in use")
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
		ID:        uuid.New().String(),
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

			app.serverError(c.Writer, c.Request, err)
		}
	})
	err = response.JSON(c.Writer, http.StatusCreated, envelope{"activation_token": activationToken.Plaintext})
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
		return
	}
}

func (app *application) ActivateUser(c *gin.Context) {
	user := app.userVerification(c)
	if user == nil {
		return
	}

	params := (&auth.UserToCreate{}).
		Email(user.Email).
		EmailVerified(true).
		Password(user.Password).
		DisplayName(user.Username).
		PhotoURL("https://cdn2.iconfinder.com/data/icons/random-outline-3/48/random_14-512.png").
		Disabled(false)

	newUserRecord, err := app.firebaseClient.CreateUser(context.Background(), params)
	if err != nil {
		if err.Error() == "user with the provided email already exists" {
			app.badRequest(c.Writer, c.Request, err)
		} else {
			fmt.Println("Error creating new user:", err)
			app.serverError(c.Writer, c.Request, err)
		}
		return
	}

	// Delete old user record and add new user record to database
	err = app.models.Users.Delete(user.ID)
	if err != nil {
		fmt.Println("Error deleting user:", err)
		app.serverError(c.Writer, c.Request, err)
		return
	}

	user.ID = newUserRecord.UID
	user.Activated = true
	fmt.Println("User:", user)
	err = app.models.Users.Insert(user)
	if err != nil {
		fmt.Println("Error inserting user:", err)
		app.serverError(c.Writer, c.Request, err)
		return
	}

	err = response.JSON(c.Writer, http.StatusCreated, nil)
	if err != nil {
		fmt.Println("Error sending response:", err)
		app.serverError(c.Writer, c.Request, err)
	}
}

func (app *application) Login(c *gin.Context) {
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

	err = response.JSON(c.Writer, http.StatusOK, nil)
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
	}
}

func (app *application) GoogleLogin(c *gin.Context) {
	var input struct {
		FirebaseID string              `json:"firebase_id"`
		IDToken    string              `json:"id_token"`
		Validator  validator.Validator `json:"-"`
	}

	err := request.DecodeJSON(c.Writer, c.Request, &input)
	if err != nil {
		app.badRequest(c.Writer, c.Request, err)
		return
	}

	oauth2Service, err := oauth2.NewService(context.Background(), option.WithoutAuthentication())

	if err != nil {
		app.serverError(c.Writer, c.Request, err)
	}

	userInfoService := oauth2.NewUserinfoV2MeService(oauth2Service)
	userInfo, err := userInfoService.Get().Do(googleapi.QueryParameter("access_token", input.IDToken))
	if err != nil {
		// e, _ := err.(*googleapi.Error)
		// fmt.Println(e.Message)
		app.serverError(c.Writer, c.Request, err)
	}
	fmt.Println(userInfo.Email)

	var user *models.User
	user, err = app.models.Users.GetByEmail(userInfo.Email)
	if err != nil && !errors.Is(err, models.ErrRecordNotFound) {
		fmt.Println("error", err)
		app.serverError(c.Writer, c.Request, err)
		return
	}
	if user == nil {
		user = &models.User{
			ID:        input.FirebaseID,
			Username:  userInfo.Name,
			Email:     userInfo.Email,
			ISS:       "google",
			SUB:       userInfo.Id,
			Activated: true,
		}

		err = app.models.Users.Insert(user)
		if err != nil {
			fmt.Println("error insert", err)
			app.serverError(c.Writer, c.Request, err)
			return
		}
		// Add the "posts:read" permission for the new user.
		err = app.models.Permissions.AddForUser(user.ID, "posts:read")
		if err != nil {
			fmt.Println("error add permission", err)
			app.serverError(c.Writer, c.Request, err)
			return
		}
	}

	err = response.JSON(c.Writer, http.StatusOK, nil)
	if err != nil {
		fmt.Println("error response", err)
		app.serverError(c.Writer, c.Request, err)
	}
}

func (app *application) Logout(c *gin.Context) {
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
			"resetLink": fmt.Sprintf("%s/reset-forgot-password/%s", app.config.FRONTEND_URL, token.Plaintext),
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
			app.invalidAuthenticationToken(c.Writer, c.Request)
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
	q := c.Request.URL.Query()

	_, err := app.checkEmailHelper(q.Get("email"))
	if err != nil {
		switch {
		case errors.Is(err, models.ErrRecordNotFound):
			err = response.JSON(c.Writer, http.StatusOK, envelope{"exists": false})
			if err != nil {
				app.serverError(c.Writer, c.Request, err)
			}
		default:
			app.serverError(c.Writer, c.Request, err)
		}
		return
	}

	err = response.JSON(c.Writer, http.StatusOK, envelope{"exists": true})
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
	}
}

func (app *application) checkUsername(c *gin.Context) {
	q := c.Request.URL.Query()

	_, err := app.checkUsernameHelper(q.Get("username"))
	if err != nil {
		switch {
		case errors.Is(err, models.ErrRecordNotFound):
			err = response.JSON(c.Writer, http.StatusOK, envelope{"exists": false})
			if err != nil {
				app.serverError(c.Writer, c.Request, err)
			}
		default:
			app.serverError(c.Writer, c.Request, err)
		}
		return
	}

	err = response.JSON(c.Writer, http.StatusOK, envelope{"exists": true})
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
