package main

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"offerland.cc/internal/models"
	"offerland.cc/internal/request"
	"offerland.cc/internal/response"
)

func (app *application) createResult(c *gin.Context) {
	// Get the user ID from the request context
	user := app.contextGetUser(c.Request)
	if user == nil {
		return
	}
	var input struct {
		AdmittedSchools []models.Result `json:"admitted_schools"`
		RejectedSchools []models.Result `json:"rejected_schools"`
	}

	err := request.DecodeJSON(c.Writer, c.Request, &input)
	if err != nil {
		app.badRequest(c.Writer, c.Request, err)
		return
	}

	// Delete all results for this user
	err = app.models.Results.Delete(user.ID)
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
		return
	}

	// Insert all results for this user
	for _, admittedSchool := range input.AdmittedSchools {
		err = app.models.Results.Insert(user.ID, admittedSchool.SchoolName, admittedSchool.MajorName, admittedSchool.AnnounceDate, "admitted", admittedSchool.Others)
		if err != nil {
			app.serverError(c.Writer, c.Request, err)
			return
		}
	}

	for _, rejectedSchool := range input.RejectedSchools {
		err = app.models.Results.Insert(user.ID, rejectedSchool.SchoolName, rejectedSchool.MajorName, rejectedSchool.AnnounceDate, "rejected", rejectedSchool.Others)
		if err != nil {
			app.serverError(c.Writer, c.Request, err)
			return
		}
	}

	err = response.JSON(c.Writer, http.StatusCreated, nil)
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
		return
	}
}

func (app *application) getUserResults(c *gin.Context) {
	viewUsername := c.Param("username")

	viewUser, err := app.models.Users.GetByUsername(viewUsername)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrRecordNotFound):
			app.notFound(c.Writer, c.Request)
		default:
			app.serverError(c.Writer, c.Request, err)
		}
		return
	}

	results, err := app.models.Results.Get(viewUser.ID)
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
		return
	}

	admittedSchools := []models.Result{}
	rejectedSchools := []models.Result{}

	for _, result := range results {
		if result.Status == "admitted" {
			admittedSchools = append(admittedSchools, result)
		} else if result.Status == "rejected" {
			rejectedSchools = append(rejectedSchools, result)
		}
	}

	err = response.JSON(c.Writer, http.StatusOK, envelope{"admitted_schools": admittedSchools, "rejected_schools": rejectedSchools})
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
		return
	}
}

func (app *application) getAllResults(c *gin.Context) {
	results, err := app.models.Results.GetAll()
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
		return
	}

	// Group results by user
	resultsByUser := map[uuid.UUID][]models.Result{}

	for _, result := range results {
		resultsByUser[result.UserID] = append(resultsByUser[result.UserID], result)
	}

	err = response.JSON(c.Writer, http.StatusOK, envelope{"results": resultsByUser})
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
		return
	}
}
