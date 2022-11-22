package main

// import (
// 	"net/http"

// 	"github.com/gin-gonic/gin"
// 	"offerland.cc/internal/response"
// )

// func (app *application) getMajors(c *gin.Context) {
// 	majors, err := app.models.Majors.GetAll()
// 	if err != nil {
// 		app.serverError(c.Writer, c.Request, err)
// 		return
// 	}

// 	err = response.JSON(c.Writer, http.StatusOK, envelope{"majors": majors})
// 	if err != nil {
// 		app.serverError(c.Writer, c.Request, err)
// 		return
// 	}
// }

// func (app *application) getMajorsBySchool(c *gin.Context) {
// 	schoolName := c.Param("school")
// 	majors, err := app.models.Majors.GetMajorsBySchool(schoolName)
// 	if err != nil {
// 		app.serverError(c.Writer, c.Request, err)
// 		return
// 	}

// 	err = response.JSON(c.Writer, http.StatusOK, envelope{"majors": majors})
// 	if err != nil {
// 		app.serverError(c.Writer, c.Request, err)
// 		return
// 	}
// }
