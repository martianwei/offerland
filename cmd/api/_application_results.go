package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"offerland.cc/internal/models"
	"offerland.cc/internal/request"
	"offerland.cc/internal/response"
)

func (app *application) CreateApplicationResult(c *gin.Context) {
	user := app.contextGetUser(c.Request)
	if user == models.AnonymousUser {
		return
	}
	var input struct {
		MajorID uuid.UUID         `json:"major_id"`
		Status  models.StatusType `json:"status"`
	}

	err := request.DecodeJSON(c.Writer, c.Request, &input)
	if err != nil {
		app.badRequest(c.Writer, c.Request, err)
		return
	}

	apr := &models.ApplicationResult{
		UserID:  user.ID,
		MajorID: input.MajorID,
		Status:  input.Status,
	}

	err = app.models.ApplicationResults.Upsert(apr)
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
		return
	}
}

func (app *application) UpdateApplicationResult(c *gin.Context) {
	user := app.contextGetUser(c.Request)
	if user == models.AnonymousUser {
		return
	}
	var input struct {
		MajorID uuid.UUID         `json:"major_id"`
		Status  models.StatusType `json:"status"`
	}

	err := request.DecodeJSON(c.Writer, c.Request, &input)
	if err != nil {
		app.badRequest(c.Writer, c.Request, err)
		return
	}

	apr := &models.ApplicationResult{
		UserID:  user.ID,
		MajorID: input.MajorID,
		Status:  input.Status,
	}

	err = app.models.ApplicationResults.Upsert(apr)
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
		return
	}
}

func (app *application) DeleteApplicationResult(c *gin.Context) {
	user := app.contextGetUser(c.Request)
	if user == models.AnonymousUser {
		return
	}
	var input struct {
		MajorID uuid.UUID `json:"major_id"`
	}

	err := request.DecodeJSON(c.Writer, c.Request, &input)
	if err != nil {
		app.badRequest(c.Writer, c.Request, err)
		return
	}

	apr := &models.ApplicationResult{
		UserID:  user.ID,
		MajorID: input.MajorID,
	}

	err = app.models.ApplicationResults.Delete(apr)
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
		return
	}
}

func (app *application) GetApplicationResults(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		app.badRequest(c.Writer, c.Request, err)
		return
	}
	user := app.contextGetUser(c.Request)
	if user == models.AnonymousUser {
		return
	}

	aprs, err := app.models.ApplicationResults.GetApplicationResultsByUserID(userID)
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
		return
	}
	err = response.JSON(c.Writer, http.StatusOK, aprs)
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
	}
}


