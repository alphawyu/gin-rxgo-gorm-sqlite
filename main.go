package main

import (
	"os"

	"com/realworld/ginrxgogorm/app"
	api "com/realworld/ginrxgogorm/app/config"
	"com/realworld/ginrxgogorm/handler"
	"com/realworld/ginrxgogorm/repository"

	log "github.com/sirupsen/logrus"

	"gorm.io/gorm"
)

// init functions run when the package is initialized.
// This lets us set configuration before any other function runs
func init() {
	logLevel, err := log.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		logLevel = log.InfoLevel
	}
	log.SetLevel(logLevel)
	logFormat := os.Getenv("LOG_FORMAT")
	switch logFormat {
	case "human":
		log.SetFormatter(
			&log.TextFormatter{
				ForceColors:            true,
				FullTimestamp:          true,
				DisableLevelTruncation: true,
			},
		)
	default:
		log.SetFormatter(&log.JSONFormatter{})
	}
}

func main() {
	log.Printf("Server started")

	db := api.NewDatabase()

	defer api.CloseDB(db)

	Migrate(db)

	ur := repository.NewUsersRepository(db)
	ar := repository.NewArticleRepository(ur, db)
	uh := handler.NewUsersHandler(ur)
	ah := handler.NewArticleHandler(ur, ar)
	a := &app.App{
		UsersRepo:      ur,
		UsersHandler:   uh,
		ArticleHandler: ah,
	}
	r := a.SetupRouter()

	r.Run() // listen and serve on 0.0.0.0:8080
}

func Migrate(db *gorm.DB) {
	db.AutoMigrate(&repository.UserModel{})
	db.AutoMigrate(&repository.ArticleModel{})
	db.AutoMigrate(&repository.TagModel{})
	db.AutoMigrate(&repository.FavoriteModel{})
	db.AutoMigrate(&repository.ArticleUserModel{})
	db.AutoMigrate(&repository.CommentModel{})
}
