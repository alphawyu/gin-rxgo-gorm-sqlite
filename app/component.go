package app

import (
	"com/realworld/ginrxgogorm/handler"
	"com/realworld/ginrxgogorm/repository"
)

type App struct {
	UsersRepo      repository.UsersRepository
	UsersHandler   handler.UsersHandler
	ArticleHandler handler.ArticleHandler
}
