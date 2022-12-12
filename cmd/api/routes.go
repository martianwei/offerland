package main

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func (app *application) SetupRouter() *gin.Engine {
	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "https://offerland.cc"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	// If user is authenticated, set the user in the context.
	// If user is not authenticated, set the user to AnonymousUser.
	router.Use(app.authenticate)

	// Return pong if the server is up.
	router.GET("/ping", app.pong)

	// If context has user, return user.
	// If context has no user, return AnonymousUser.
	router.GET("/whoami", app.whoAmI)

	auth := router.Group("/auth")
	{
		// Sign up
		auth.POST("/signup", app.Signup)
		// Activate account
		auth.POST("/activate/:token", app.Activate)
		// Check if the username is already in use
		auth.GET("/check_username/:username", app.checkUsername)
		// Check if the email address is already in use
		auth.GET("/check_email/:email", app.checkEmail)
		auth.GET("/check_author/:authorname", app.checkAuthor)
		// Login
		auth.POST("/login", app.Login)
		auth.POST("/googlelogin", app.GoogleLogin)
		// Logout
		auth.POST("/logout", app.Logout)
		// Refresh token
		auth.POST("/refresh_token", app.refreshToken)
	}
	// Forgot password
	router.POST("/forgot-password", app.userForgotPassword)
	router.POST("/reset-forgot-password/:token", app.userForgotPasswordReset)

	result := router.Group("/results")
	{
		result.POST("", app.createResult)
		result.GET("/:username", app.getUserResults)
		result.GET("/all", app.getAllResults)
	}

	// _api := router.Group("/_api")
	// {
	// 	_api.GET("/schools", app.getSchools)
	// 	_api.GET("/majors", app.getMajors)
	// 	_api.GET("/majors/:school", app.getMajorsBySchool)
	// }

	// user := router.Group("/users")
	// {
	// 	user.GET("/:id", app.GetUser)
	// }
	// postsHandler := handler.NewPostHandler(app.db)
	post := router.Group("/posts")
	{
		post.GET("/:id", app.GetPost)
		post.GET("", app.GetAllPosts)
		post.POST("", app.CreatePost)
		post.PUT("/:id", app.UpdatePost)
		post.DELETE("/:id", app.DeletePost)
	}

	// application_result := router.Group("/application_results")
	// {
	// 	application_result.GET("/:user_id", app.GetApplicationResults)
	// 	application_result.POST("/", app.CreateApplicationResult)
	// 	application_result.PUT("/", app.UpdateApplicationResult)
	// 	application_result.DELETE("/",app.DeleteApplicationResult)
	// }
	// school := router.Group("/school")
	// {
	// 	school.GET("", app.getSchools)
	// }

	return router
}
