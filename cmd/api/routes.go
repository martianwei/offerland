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

	user := router.Group("/user")
	{
		user.GET("/check", app.userCheck)
		// user.GET("/:user_id", usersHandler.GetUser)
		user.POST("/signup", app.userSignup)
		user.POST("/activate/:token", app.userActivate)
		user.POST("/googlelogin", app.UserSignupWithGoogle)
		user.POST("/login", app.userLogin)
		// user.POST("/forgot-password", usersHandler.UserForgotPassword)
		// user.POST("/reset-password/:token", usersHandler.UserResetPassword)
	}

	return router
}
