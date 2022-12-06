package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"offerland.cc/internal/models"
	"offerland.cc/internal/request"
	"offerland.cc/internal/response"
)

func (app *application) CreatePost(c *gin.Context) {
	user := app.contextGetUser(c.Request)
	if user == models.AnonymousUser {
		return
	}
	var input struct {
		Title     string `json:"title" binding:"required"`
		Body      string `json:"body" binding:"required"`
		AddResult bool   `json:"add_result" binding:"required"`
	}

	err := request.DecodeJSON(c.Writer, c.Request, &input)
	if err != nil {
		app.badRequest(c.Writer, c.Request, err)
		return
	}

	post := &models.Post{
		PostID:    uuid.New(),
		Title:     input.Title,
		AddResult: input.AddResult,
		Body:      input.Body,
		UserID:    user.ID,
	}

	err = app.models.Posts.Upsert(post)
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
		return
	}
}

func (app *application) UpdatePost(c *gin.Context) {
	user := app.contextGetUser(c.Request)
	if user == models.AnonymousUser {
		return
	}
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		app.badRequest(c.Writer, c.Request, err)
		return
	}
	var input struct {
		Title     string `json:"title" binding:"required"`
		Body      string `json:"body" binding:"required"`
		AddResult bool   `json:"add_result" binding:"required"`
	}

	err = request.DecodeJSON(c.Writer, c.Request, &input)
	if err != nil {
		app.badRequest(c.Writer, c.Request, err)
		return
	}

	post := &models.Post{
		PostID:    postID,
		Title:     input.Title,
		AddResult: input.AddResult,
		Body:      input.Body,
		UserID:    user.ID,
	}
	result, err := app.models.Posts.CheckPostIsMine(post)
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
		return
	} else if !result {
		app.inactiveAccount(c.Writer, c.Request)
		return
	}
	err = app.models.Posts.Upsert(post)
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
		return
	}

}

func (app *application) DeletePost(c *gin.Context) {
	user := app.contextGetUser(c.Request)
	if user == models.AnonymousUser {
		return
	}

	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		app.badRequest(c.Writer, c.Request, err)
		return
	}

	post := &models.Post{
		PostID: postID,
		UserID: user.ID,
	}

	result, err := app.models.Posts.CheckPostIsMine(post)
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
		return
	} else if !result {
		app.inactiveAccount(c.Writer, c.Request)
		return
	}
	err = app.models.Posts.Delete(post)
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
		return
	}

}

func (app *application) GetPost(c *gin.Context) {
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		app.badRequest(c.Writer, c.Request, err)
		return
	}
	user := app.contextGetUser(c.Request)
	if user == models.AnonymousUser {
		return
	}
	if err != nil {
		app.badRequest(c.Writer, c.Request, err)
		return
	}

	result, err := app.models.Posts.GetPostByID(postID)
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
		return
	}
	err = response.JSON(c.Writer, http.StatusOK, result)
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
	}
}

func (app *application) GetAllPosts(c *gin.Context) {
	user := app.contextGetUser(c.Request)
	if user == models.AnonymousUser {
		return
	}

	posts, err := app.models.Posts.GetAllPosts()
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
		return
	}
	err = response.JSON(c.Writer, http.StatusOK, posts)
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
	}
}
