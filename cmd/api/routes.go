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
		auth.POST("/signup", app.userSignup)
		// Activate account
		auth.POST("/activate/:token", app.userActivate)
		// Check if the username is already in use
		auth.GET("/check_username/:username", app.checkUsername)
		// Check if the email address is already in use
		auth.GET("/check_email/:email", app.checkEmail)
		// Login
		auth.POST("/login", app.userLogin)
		auth.POST("/googlelogin", app.userGoogleLogin)
		// Logout
		auth.POST("/logout", app.userLogout)
	}
	// Forgot password
	router.POST("/forgot-password", app.userForgotPassword)
	router.POST("/reset-forgot-password/:token", app.userForgotPasswordReset)

	result := router.Group("/results")
	{
		result.POST("", app.createResult)
		result.GET("", app.getUserResults)
		result.GET("/all", app.getAllResults)
	}

	// _api := router.Group("/_api")
	// {
	// 	_api.GET("/schools", app.getSchools)
	// 	_api.GET("/majors", app.getMajors)
	// 	_api.GET("/majors/:school", app.getMajorsBySchool)
	// }

	// user := router.Group("/user")
	// {
	// 	user.GET("/:user_id", usersHandler.GetUser)
	// }
	// postsHandler := handler.NewPostHandler(app.db)
	// post := router.Group("/post")
	// {
	// 	post.GET("/:post_id", postsHandler.GetPost)
	// 	post.GET("/", postsHandler.GetAllPosts)
	// 	post.POST("/", postsHandler.CreatePost)
	// }
	// school := router.Group("/school")
	// {
	// 	school.GET("", app.getSchools)
	// }

	return router
}
