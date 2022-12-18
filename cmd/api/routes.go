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

	// Return pong if the server is up.
	router.GET("/ping", app.pong)

	// If context has user, return user.
	// If context has no user, return AnonymousUser.
	router.GET("/whoami", app.authenticate, app.whoAmI)

	auth := router.Group("/auth")
	{
		auth.POST("/signup", app.Signup)
		auth.POST("/activate/:token", app.Activate)

		auth.GET("/check_username/:username", app.checkUsername)
		auth.GET("/check_email/:email", app.checkEmail)
		auth.GET("/check_author/:authorname", app.checkAuthor)

		auth.POST("/login", app.Login)
		auth.POST("/googlelogin", app.GoogleLogin)

		auth.POST("/logout", app.Logout)

		auth.POST("/refresh_token", app.refreshToken)
	}

	router.POST("/forgot-password", app.userForgotPassword)
	router.POST("/reset-forgot-password/:token", app.userForgotPasswordReset)

	result := router.Group("/results")
	{
		result.POST("", app.authenticate, app.createResult)
		result.GET("/:username", app.authenticate, app.getUserResults)
		result.GET("", app.authenticate, app.getAllResults)
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
		post.POST("", app.authenticate, app.CreatePost)
		post.PUT("/:id", app.authenticate, app.UpdatePost)
		post.DELETE("/:id", app.authenticate, app.DeletePost)
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
