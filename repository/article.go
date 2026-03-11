package repository

import (
	"errors"
	"strconv"

	log "github.com/sirupsen/logrus"

	"gorm.io/gorm"
)

type ArticleModel struct {
	gorm.Model
	Slug        string `gorm:"unique_index"`
	Title       string
	Description string `gorm:"size:2048"`
	Body        string `gorm:"size:2048"`
	Author      ArticleUserModel
	AuthorID    uint
	Tags        []TagModel     `gorm:"many2many:article_tags;"`
	Comments    []CommentModel `gorm:"ForeignKey:ArticleID"`
}

// ArticleUserModel represents articleUser in Article domain. It is not "necessary",
//	but a very useful design detail to keep UserModel separate from Article
type ArticleUserModel struct {
	gorm.Model
	UserModel      UserModel
	UserModelID    uint
	ArticleModels  []ArticleModel  `gorm:"ForeignKey:AuthorID"`
	FavoriteModels []FavoriteModel `gorm:"ForeignKey:FavoriteByID"`
}

type FavoriteModel struct {
	gorm.Model
	Favorite     ArticleModel
	FavoriteID   uint
	FavoriteBy   ArticleUserModel
	FavoriteByID uint
}

type TagModel struct {
	gorm.Model
	Tag           string         `gorm:"unique_index"`
	ArticleModels []ArticleModel `gorm:"many2many:article_tags;"`
}

type CommentModel struct {
	gorm.Model
	Article   ArticleModel
	ArticleID uint
	Author    ArticleUserModel
	AuthorID  uint
	Body      string `gorm:"size:2048"`
}

//go:generate mockgen -destination mock/article_repo_mock.go -source article.go ArticleRepository
type ArticleRepository interface {
	GetArticleUserModel(userModel UserModel) ArticleUserModel

	SaveOne(data interface{}) error
	Update(model *ArticleModel) error
	Delete(condition interface{}) error

	FavoritesCount(article ArticleModel) uint
	IsFavoriteBy(article ArticleModel, articleUser ArticleUserModel) bool
	FavoriteBy(article ArticleModel, articleUser ArticleUserModel) error
	UnfavoriteBy(article ArticleModel, articleUser ArticleUserModel) error

	FindOneArticle(condition interface{}) (ArticleModel, error)
	FindManyArticle(tag, author, limit, offset, favorited string) ([]ArticleModel, int, error)
	GetArticleFeed(self ArticleUserModel, limit, offset string) ([]ArticleModel, int, error)

	GetComments(self *ArticleModel) error

	SetTags(model *ArticleModel, tags []string) error
	GetAllTags() ([]TagModel, error)

	GatherLoginUserStat(ArticleUserModel, ArticleModel) (bool, bool, uint, ArticleModel)
}

type ArticleRepositoryImpl struct {
	db *gorm.DB
	ur UsersRepository
}

func NewArticleRepository(ur UsersRepository, db *gorm.DB) *ArticleRepositoryImpl {
	return &ArticleRepositoryImpl{
		db: db,
		ur: ur,
	}
}

// to ensure the __impl implements the interface at compile-time
var _ ArticleRepository = &ArticleRepositoryImpl{}

func (r *ArticleRepositoryImpl) GetArticleUserModel(userModel UserModel) ArticleUserModel {
	var articleUserModel ArticleUserModel
	if userModel.ID == 0 {
		return articleUserModel
	}
	r.db.Where(&ArticleUserModel{
		UserModelID: userModel.ID,
	}).FirstOrCreate(&articleUserModel)
	articleUserModel.UserModel = userModel
	return articleUserModel
}

func (r *ArticleRepositoryImpl) FavoritesCount(article ArticleModel) uint {
	var count int64
	r.db.Model(&FavoriteModel{}).Where(FavoriteModel{
		FavoriteID: article.ID,
	}).Count(&count)
	return uint(count)
}

func (r *ArticleRepositoryImpl) IsFavoriteBy(article ArticleModel, articleUser ArticleUserModel) bool {
	var favorite FavoriteModel
	r.db.Where(FavoriteModel{
		FavoriteID:   article.ID,
		FavoriteByID: articleUser.ID,
	}).First(&favorite)
	return favorite.ID != 0
}

func (r *ArticleRepositoryImpl) FavoriteBy(article ArticleModel, articleUser ArticleUserModel) error {
	var favorite FavoriteModel
	err := r.db.FirstOrCreate(&favorite, &FavoriteModel{
		FavoriteID:   article.ID,
		FavoriteByID: articleUser.ID,
	}).Error
	if err != nil {
		log.Errorf("FavoriteBy (gorm:FirstOrCreate) error %v", err)
	}
	return err
}

func (r *ArticleRepositoryImpl) UnfavoriteBy(article ArticleModel, articleUser ArticleUserModel) error {
	err := r.db.Where(FavoriteModel{
		FavoriteID:   article.ID,
		FavoriteByID: articleUser.ID,
	}).Delete(&FavoriteModel{}).Error
	if err != nil {
		log.Errorf("UnfavoriteBy (gorm:Delete) error %v", err)
	}
	return err
}

func (r *ArticleRepositoryImpl) SaveOne(data interface{}) error {
	err := r.db.Save(data).Error
	if err != nil {
		log.Errorf("SaveOne (gorm:Save) error %v", err)
	}
	return err
}

func (r *ArticleRepositoryImpl) FindOneArticle(condition interface{}) (ArticleModel, error) {
	var model ArticleModel
	tx := r.db.Begin()
	tx.Where(condition).Preload("Author").Preload("Author.UserModel").Preload("Tags").First(&model)
	err := tx.Commit().Error

	if err != nil {
		log.Errorf("FindOneArticle (gorm:Where,Preload,Fist) error %v", err)
	}
	if err == nil && model.ID == 0 {
		err = errors.New("not found")
		log.Error("FindOneArticle not found")
	}

	return model, err
}

func (r *ArticleRepositoryImpl) GetComments(self *ArticleModel) error {
	tx := r.db.Begin()
	tx.Model(self).Preload("Comments").First(self)
	for i, _ := range self.Comments {
		tx.Model(&self.Comments[i]).Preload("Author").Preload("Author.UserModel").First(&self.Comments[i])
	}
	err := tx.Commit().Error
	if err != nil {
		log.Errorf("GetComment (gorm:Model,Preload,Fist) error %v", err)
	}
	return err
}

func (r *ArticleRepositoryImpl) GetAllTags() ([]TagModel, error) {
	var models []TagModel
	err := r.db.Find(&models).Error
	if err != nil {
		log.Errorf("GetAllTags (gorm:Find) error %v", err)
	}
	return models, err
}

func (r *ArticleRepositoryImpl) FindManyArticle(tag, author, limit, offset, favorited string) ([]ArticleModel, int, error) {
	var models []ArticleModel
	var count int64

	offset_int, err := strconv.Atoi(offset)
	if err != nil {
		offset_int = 0
	}

	limit_int, err := strconv.Atoi(limit)
	if err != nil {
		limit_int = 20
	}

	tx := r.db.Begin()
	if tag != "" {
		var tagModel TagModel
		tx.Where(TagModel{Tag: tag}).First(&tagModel)
		if tagModel.ID != 0 {
			tx.Model(&tagModel).Preload("ArticleModels", func(db *gorm.DB) *gorm.DB {
				return db.Offset(offset_int).Limit(limit_int).Order("created_at DESC")
			}).First(&tagModel)
			models = tagModel.ArticleModels
			count = tx.Model(&tagModel).Association("ArticleModels").Count()
		}
	} else if author != "" {
		var userModel UserModel
		tx.Where(UserModel{Username: author}).First(&userModel)
		articleUserModel := r.GetArticleUserModel(userModel)
		if articleUserModel.ID != 0 {
			count = tx.Model(&articleUserModel).Association("ArticleModels").Count()
			tx.Model(&articleUserModel).Preload("ArticleModels", func(db *gorm.DB) *gorm.DB {
				return db.Offset(offset_int).Limit(limit_int).Order("created_at DESC")
			}).First(&articleUserModel)
			models = articleUserModel.ArticleModels
		}
	} else if favorited != "" {
		var userModel UserModel
		tx.Where(UserModel{Username: favorited}).First(&userModel)
		articleUserModel := r.GetArticleUserModel(userModel)
		if articleUserModel.ID != 0 {
			var favoriteModels []FavoriteModel
			tx.Where(FavoriteModel{
				FavoriteByID: articleUserModel.ID,
			}).Offset(offset_int).Limit(limit_int).Find(&favoriteModels)

			count = tx.Model(&articleUserModel).Association("FavoriteModels").Count()
			for _, favorite := range favoriteModels {
				var model ArticleModel
				tx.Model(&favorite).Preload("Favorite", &model)
				models = append(models, model)
			}
		}
	} else {
		r.db.Model(&models).Count(&count)
		r.db.Offset(offset_int).Limit(limit_int).Find(&models)
	}

	for i := range models {
		tx.Model(&models[i]).Preload("Author").Preload("Author.UserModel").Preload("Tags").First(&models[i])
	}
	err = tx.Commit().Error
	if err != nil {
		log.Errorf("FindManyArticle (gorm:Where,Preload,Fist) error %v", err)
	}
	return models, int(count), err
}

func (r *ArticleRepositoryImpl) GetArticleFeed(self ArticleUserModel, limit, offset string) ([]ArticleModel, int, error) {
	var models []ArticleModel
	var count int64

	offset_int, err := strconv.Atoi(offset)
	if err != nil {
		offset_int = 0
	}
	limit_int, err := strconv.Atoi(limit)
	if err != nil {
		limit_int = 20
	}

	tx := r.db.Begin()
	followings, err := r.ur.GetFollowings(self.UserModel)
	if err != nil {
		return models, int(0), err
	}
	var authorIds []uint
	for _, following := range followings {
		articleUserModel := r.GetArticleUserModel(following)
		authorIds = append(authorIds, articleUserModel.ID)
	}

	tx.Model(&ArticleModel{}).Where("author_id in (?)", authorIds).Find(&models).Count(&count)
	tx.Where("author_id in (?)", authorIds).Order("updated_at desc").Offset(offset_int).Limit(limit_int).Find(&models)

	for i := range models {
		tx.Model(&models[i]).Preload("Author").Preload("Author.UserModel").Preload("Tags").First(&models[i])
	}
	err = tx.Commit().Error
	if err != nil {
		log.Errorf("GetArticleFeed (gorm:Where,Preload,Fist) error %v", err)
	}
	return models, int(count), err
}

func (r *ArticleRepositoryImpl) SetTags(model *ArticleModel, tags []string) error {
	var tagList []TagModel
	for _, tag := range tags {
		var tagModel TagModel
		err := r.db.FirstOrCreate(&tagModel, TagModel{Tag: tag}).Error
		if err != nil {
			log.Errorf("SetTags (gorm:FistOrCreate) error %v", err)
			return err
		}
		tagList = append(tagList, tagModel)
	}
	model.Tags = tagList
	return nil
}

func (r *ArticleRepositoryImpl) Update(model *ArticleModel) error {
	err := r.db.Save(model).Error
	if err != nil {
		log.Errorf("Update (gorm:Save) error %v", err)
	}
	return err
}

func (r *ArticleRepositoryImpl) Delete(condition interface{}) error {
	err := r.db.Where(condition).Delete(&condition).Error
	if err != nil {
		log.Errorf("Delete (%T gorm:Delete) error %v", condition, err)
	}
	return err
}

func (r *ArticleRepositoryImpl) GatherLoginUserStat(loginUser ArticleUserModel,
	articleM ArticleModel) (followingFlag, favorite bool, favoriteCount uint, am ArticleModel) {
	// NOTE" IsFollowing only need the author user id for the look up
	followingFlag, err := r.ur.IsFollowings(loginUser.UserModel, articleM.Author.UserModel)
	if err != nil {
		log.Infof("Ignore IsFollowing error %v", err)
		followingFlag = false // default (unknown) is not following
	}
	favorite = r.IsFavoriteBy(articleM, loginUser)
	favoriteCount = r.FavoritesCount(articleM)
	am = articleM
	return
}
