package main

// import (
// 	"net/http"

// 	"github.com/gin-gonic/gin"
// 	"offerland.cc/internal/response"
// )

// func (app *application) getSchools(c *gin.Context) {
// 	schools, err := app.models.Schools.GetAll()
// 	if err != nil {
// 		app.serverError(c.Writer, c.Request, err)
// 		return
// 	}

// 	err = response.JSON(c.Writer, http.StatusOK, envelope{"schools": schools})
// 	if err != nil {
// 		app.serverError(c.Writer, c.Request, err)
// 		return
// 	}
// }
