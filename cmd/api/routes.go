package main

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func (app *application) SetupRouter() *gin.Engine {
	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://localhost:3000"},
		AllowMethods: []string{"OPTIONS", "PUT", "DELETE", "PATCH"},
		AllowHeaders: []string{"Content-Type", "Authorization"},
	}))
	router.Use(app.authenticate)

	// postsHandler := handler.NewPostHandler(app.db)
	// post := router.Group("/post")
	// {
	// 	post.GET("/:post_id", postsHandler.GetPost)
	// 	post.GET("/", postsHandler.GetAllPosts)
	// 	post.POST("/", postsHandler.CreatePost)
	// }

	router.POST("/signup", app.userSignup)
	router.POST("/activate/:token", app.userActivate)
	router.POST("/login", app.userLogin)
	router.POST("/googlelogin", app.UserSignupWithGoogle)
	router.POST("/forgot-password", app.UserForgotPassword)
	router.POST("/reset-password/:token", app.resetPassword)

	user := router.Group("/user")
	{
		user.GET("/check", app.userCheck)
		// user.GET("/:user_id", usersHandler.GetUser)

	}
	return router
}
