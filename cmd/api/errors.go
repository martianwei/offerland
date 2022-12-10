package main

import (
	"net/http"
	"strings"

	"offerland.cc/internal/response"
	"offerland.cc/internal/validator"
)

func (app *application) errorMessage(w http.ResponseWriter, r *http.Request, status int, message string, headers http.Header) {
	message = strings.ToUpper(message[:1]) + message[1:]

	err := response.JSONWithHeaders(w, status, map[string]string{"Error": message}, headers)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (app *application) serverError(w http.ResponseWriter, r *http.Request, err error) {
	app.logger.Error(err)

	message := "The server encountered a problem and could not process your request"
	app.errorMessage(w, r, http.StatusInternalServerError, message, nil)
}

func (app *application) notFound(w http.ResponseWriter, r *http.Request) {
	app.errorMessage(w, r, http.StatusNotFound, "The requested resource could not be found, please try again.", nil)
}

func (app *application) emailNotFound(w http.ResponseWriter, r *http.Request) {
	message := "The email address you entered could not be found"
	app.errorMessage(w, r, http.StatusNotFound, message, nil)
}

func (app *application) badRequest(w http.ResponseWriter, r *http.Request, err error) {
	app.errorMessage(w, r, http.StatusBadRequest, err.Error(), nil)
}

func (app *application) createConflict(w http.ResponseWriter, r *http.Request) {
	message := "unable to create the record due to a conflict, please try again"
	app.errorMessage(w, r, http.StatusConflict, message, nil)
}

func (app *application) editConflict(w http.ResponseWriter, r *http.Request) {
	message := "unable to update the record due to an edit conflict, please try again"
	app.errorMessage(w, r, http.StatusConflict, message, nil)
}

func (app *application) failedValidation(w http.ResponseWriter, r *http.Request, v validator.Validator) {
	err := response.JSON(w, http.StatusUnprocessableEntity, v)
	if err != nil {
		app.serverError(w, r, err)
	}
}

func (app *application) invalidToken(w http.ResponseWriter, r *http.Request) {
	message := "Your token is invalid or has expired, please try again"
	app.errorMessage(w, r, http.StatusUnauthorized, message, nil)
}

func (app *application) invalidCredentials(w http.ResponseWriter, r *http.Request) {
	message := "invalid email address or password"
	app.errorMessage(w, r, http.StatusUnauthorized, message, nil)
}

func (app *application) inactiveAccount(w http.ResponseWriter, r *http.Request) {
	message := "your user account must be activated to access this resource"
	app.errorMessage(w, r, http.StatusForbidden, message, nil)
}

func (app *application) expiredToken(w http.ResponseWriter, r *http.Request) {
	app.logger.Warning("expired token")
	message := "Your token has expired, please try again"
	app.errorMessage(w, r, http.StatusForbidden, message, nil)
}

func (app *application) invalidAuthenticationToken(w http.ResponseWriter, r *http.Request) {
	app.errorMessage(w, r, http.StatusUnauthorized, "Invalid or missing authentication token", nil)
}
