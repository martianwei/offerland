package main

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"offerland.cc/internal/models"
	"offerland.cc/internal/request"
	"offerland.cc/internal/response"
)

func (app *application) createResult(c *gin.Context) {
	fmt.Println("createResult")
	// Get the user ID from the request context
	user := app.contextGetUser(c.Request)
	if user == nil {
		return
	}
	var input struct {
		AdmittedSchools []models.AdmittedSchool `json:"admitted_schools"`
		RejectedSchools []models.RejectedSchool `json:"rejected_schools"`
	}

	err := request.DecodeJSON(c.Writer, c.Request, &input)
	fmt.Println("input admitted schools", input.AdmittedSchools)
	fmt.Println("input rejected schools", input.RejectedSchools)
	if err != nil {
		app.badRequest(c.Writer, c.Request, err)
		return
	}

	// Insert the result into the database
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		fmt.Println("Inserting admitted schools...")
		for _, admittedSchool := range input.AdmittedSchools {
			err = app.models.Results.Insert(user.ID, admittedSchool.SchoolName, admittedSchool.MajorName, admittedSchool.AnnounceDate, admittedSchool.Others)
			if err != nil {
				app.serverError(c.Writer, c.Request, err)
				return
			}
		}
		fmt.Println("Finished inserting admitted schools")
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		fmt.Println("Inserting rejected schools...")
		for _, rejectedSchool := range input.RejectedSchools {
			err = app.models.Results.Insert(user.ID, rejectedSchool.SchoolName, rejectedSchool.MajorName, rejectedSchool.AnnounceDate, rejectedSchool.Others)
			if err != nil {
				app.serverError(c.Writer, c.Request, err)
				return
			}
		}
		fmt.Println("Finished inserting admitted schools")
		wg.Done()
	}()

	wg.Wait()

	err = response.JSON(c.Writer, http.StatusCreated, nil)
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
		return
	}
}
