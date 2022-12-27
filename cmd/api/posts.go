package main

import (
	"errors"
	"fmt"
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
		fmt.Println("Upsert error: ", err)
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

	if err != nil {
		app.badRequest(c.Writer, c.Request, err)
		return
	}

	post, err := app.models.Posts.GetPostByID(postID)
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
		return
	}

	// Embed user in post
	user, err := app.models.Users.Get(post.UserID)
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
		return
	}
	postResponse := map[string]interface{}{
		"user": user,
		"post": post,
	}

	err = response.JSON(c.Writer, http.StatusOK, postResponse)
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
	}
}

func (app *application) GetAllPosts(c *gin.Context) {
	filter := c.Request.URL.Query()

	// Check if username in filter exists in db
	if _, ok := filter["username"]; ok {
		user, err := app.models.Users.GetByUsername(filter["username"][0])
		if err != nil {
			switch {
			case errors.Is(err, models.ErrRecordNotFound):
				app.notFound(c.Writer, c.Request)
			default:
				app.serverError(c.Writer, c.Request, err)
			}
			return
		}
		filter["user_id"] = []string{user.ID}
		delete(filter, "username")
	}

	posts, err := app.models.Posts.GetAllPosts(filter)
	// Check if user exists, if not return empty array
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
		return
	}

	// If len of posts is 0, return empty array
	if len(posts) == 0 {
		// return empty array not null
		err = response.JSON(c.Writer, http.StatusOK, []interface{}{})
		if err != nil {
			app.serverError(c.Writer, c.Request, err)
		}
		return
	}

	postsResponse := []map[string]interface{}{}
	// embed user data, dont include activated field
	for _, post := range posts {
		user, err := app.models.Users.Get(post.UserID)
		if err != nil {
			app.serverError(c.Writer, c.Request, err)
			return
		}

		// create a struct with only the fields we want to include
		postsResponse = append(postsResponse, map[string]interface{}{
			"post": post,
			"user": user,
		})
	}

	err = response.JSON(c.Writer, http.StatusOK, postsResponse)
	if err != nil {
		app.serverError(c.Writer, c.Request, err)
	}
}
