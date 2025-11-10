package app

import (
	"com/realworld/ginrxgogorm/middleware"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (a App) SetupRouter() *gin.Engine {
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"alive": true,
		})
	})

	v1 := r.Group("/api")
	a.UsersRegister(v1.Group("/users"))

	v1.Use(middleware.AuthMiddleware(a.UsersRepo, false))
	a.ArticlesAnonymousRegister(v1.Group("/articles"))
	a.TagsAnonymousRegister(v1.Group("/tags"))

	v1.Use(middleware.AuthMiddleware(a.UsersRepo, true))
	a.UserRegister(v1.Group("/user"))
	a.ProfileRegister(v1.Group("/profiles"))
	a.ArticlesRegister(v1.Group("/articles"))

	return r
}

func (a *App) UsersRegister(router *gin.RouterGroup) {
	router.POST("", a.UsersHandler.UsersRegistration)
	router.POST("/login", a.UsersHandler.UsersLogin)
}

func (a *App) UserRegister(router *gin.RouterGroup) {
	router.GET("", a.UsersHandler.UserRetrieve)
	router.PUT("", a.UsersHandler.UserUpdate)
}

func (a *App) ProfileRegister(router *gin.RouterGroup) {
	router.GET("/:username", a.UsersHandler.ProfileRetrieve)
	router.POST("/:username/follow", a.UsersHandler.ProfileFollow)
	router.DELETE("/:username/follow", a.UsersHandler.ProfileUnfollow)
}

func (a *App) ArticlesRegister(router *gin.RouterGroup) {
	router.POST("", a.ArticleHandler.ArticleCreate)
	router.PUT("/:slug", a.ArticleHandler.ArticleUpdate)
	router.DELETE("/:slug", a.ArticleHandler.ArticleDelete)
	router.POST("/:slug/favorite", a.ArticleHandler.ArticleFavorite)
	router.DELETE("/:slug/favorite", a.ArticleHandler.ArticleUnfavorite)
	router.POST("/:slug/comments", a.ArticleHandler.ArticleCommentCreate)
	router.DELETE("/:slug/comments/:id", a.ArticleHandler.ArticleCommentDelete)
}

func (a *App) ArticlesAnonymousRegister(router *gin.RouterGroup) {
	router.GET("", a.ArticleHandler.ArticleList)
	router.GET("/feed", a.ArticleHandler.ArticleFeed)
	router.GET("/:slug", a.ArticleHandler.ArticleRetrieve)
	router.GET("/:slug/comments", a.ArticleHandler.ArticleCommentList)
}

func (a *App) TagsAnonymousRegister(router *gin.RouterGroup) {
	router.GET("", a.ArticleHandler.TagList)
}
