package main

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func (app *application) SetupRouter() *gin.Engine {
	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"OPTIONS", "GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}))
	router.Use(app.authenticate)

	router.GET("/whoami", app.whoAmI)

	auth := router.Group("/auth")
	{
		auth.POST("/signup", app.userSignup)
		auth.POST("/login", app.userLogin)
		auth.POST("/googlelogin", app.userGoogleLogin)
		auth.POST("/logout", app.userLogout)
		auth.POST("/activate/:token", app.userActivate)
	}
	router.POST("/forgot-password", app.userForgotPassword)
	router.POST("/reset-password/:token", app.userResetPassword)

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
