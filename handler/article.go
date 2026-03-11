package handler

import (
	"errors"
	"net/http"
	"strconv"

	"com/realworld/ginrxgogorm/model"
	"com/realworld/ginrxgogorm/repository"
	"com/realworld/ginrxgogorm/util"

	"github.com/gin-gonic/gin"
	"github.com/gosimple/slug"
	"gorm.io/gorm"
)

//go:generate mockgen -destination mock/article_mock.go -source article_handler.go ArticleHandler
type ArticleHandler interface {
	ArticleCreate(c *gin.Context)
	ArticleUpdate(c *gin.Context)
	ArticleDelete(c *gin.Context)

	ArticleList(c *gin.Context)
	ArticleRetrieve(c *gin.Context)
	ArticleFeed(c *gin.Context)

	ArticleFavorite(c *gin.Context)
	ArticleUnfavorite(c *gin.Context)

	ArticleCommentCreate(c *gin.Context)
	ArticleCommentDelete(c *gin.Context)
	ArticleCommentList(c *gin.Context)

	TagList(c *gin.Context)
}

type ArticleHandlerImpl struct {
	UsersRepo   repository.UsersRepository
	ArticleRepo repository.ArticleRepository
}

func NewArticleHandler(usersRepo repository.UsersRepository,
	articleRepo repository.ArticleRepository) *ArticleHandlerImpl {
	return &ArticleHandlerImpl{
		UsersRepo:   usersRepo,
		ArticleRepo: articleRepo,
	}
}

// to ensure the __impl implements the interface at compile-time
var _ ArticleHandler = &ArticleHandlerImpl{}

func (a *ArticleHandlerImpl) ArticleCreate(c *gin.Context) {
	var aw model.ArticleWrapper
	if err := c.BindJSON(&aw); err != nil {
		c.JSON(http.StatusUnprocessableEntity, util.NewValidatorError(err))
		return
	}

	article := aw.Article
	loginUser := getLoginUser(c)
	loginUserArticle := a.ArticleRepo.GetArticleUserModel(loginUser)
	articleModel := repository.ArticleModel{
		Slug:        slug.Make(article.Title),
		Title:       article.Title,
		Description: article.Description,
		Body:        article.Body,
		Author:      loginUserArticle,
	}

	if err := a.ArticleRepo.SaveOne(&articleModel); err != nil {
		c.JSON(http.StatusUnprocessableEntity, util.NewError("database", err))
		return
	}

	articleResp := makeArticleResponse(a.ArticleRepo.GatherLoginUserStat(loginUserArticle, articleModel))
	c.JSON(http.StatusCreated, gin.H{"article": articleResp})
}

func (a *ArticleHandlerImpl) ArticleList(c *gin.Context) {
	tag := c.Query("tag")
	author := c.Query("author")
	favorited := c.Query("favorited")
	limit := c.Query("limit")
	offset := c.Query("offset")

	articleModels, modelCount, err := a.ArticleRepo.FindManyArticle(tag, author, limit, offset, favorited)
	if err != nil {
		c.JSON(http.StatusNotFound, util.NewError("articles", errors.New("invalid param")))
		return
	}

	loginUser := getLoginUser(c)
	loginUserArticle := a.ArticleRepo.GetArticleUserModel(loginUser)
	response := []model.ArticleResponse{}
	for _, article := range articleModels {
		response = append(response, makeArticleResponse(a.ArticleRepo.GatherLoginUserStat(loginUserArticle, article)))
	}
	c.JSON(http.StatusOK, gin.H{"articles": response, "articlesCount": modelCount})
}

func (a *ArticleHandlerImpl) ArticleFeed(c *gin.Context) {
	limit := c.Query("limit")
	offset := c.Query("offset")

	loginUser := getLoginUser(c)
	if loginUser.ID == 0 {
		c.JSON(http.StatusNotFound, util.NewError("articles", errors.New("Require Current Login User!")))
		return
	}
	loginUserArticle := a.ArticleRepo.GetArticleUserModel(loginUser)
	articleModels, modelCount, err := a.ArticleRepo.GetArticleFeed(loginUserArticle, limit, offset)
	if err != nil {
		c.JSON(http.StatusNotFound, util.NewError("articles", errors.New("invalid param")))
		return
	}

	response := []model.ArticleResponse{}
	for _, article := range articleModels {
		response = append(response, makeArticleResponse(a.ArticleRepo.GatherLoginUserStat(loginUserArticle, article)))
	}
	c.JSON(http.StatusOK, gin.H{"articles": response, "articlesCount": modelCount})
}

func (a *ArticleHandlerImpl) ArticleRetrieve(c *gin.Context) {
	slug := c.Param("slug")
	articleModel, err := a.ArticleRepo.FindOneArticle(&repository.ArticleModel{Slug: slug})
	if err != nil {
		c.JSON(http.StatusNotFound, util.NewError("articles", errors.New("invalid slug")))
		return
	}

	loginUser := getLoginUser(c)
	loginUserArticle := a.ArticleRepo.GetArticleUserModel(loginUser)
	response := makeArticleResponse(a.ArticleRepo.GatherLoginUserStat(loginUserArticle, articleModel))
	c.JSON(http.StatusOK, gin.H{"article": response})
}

func (a *ArticleHandlerImpl) ArticleUpdate(c *gin.Context) {
	var uaw model.UpdateArticleWrapper
	if err := c.BindJSON(&uaw); err != nil {
		c.JSON(http.StatusUnprocessableEntity, util.NewValidatorError(err))
		return
	}

	pSlug := c.Param("slug")
	articleModel, err := a.ArticleRepo.FindOneArticle(&repository.ArticleModel{Slug: pSlug})
	if err != nil {
		c.JSON(http.StatusNotFound, util.NewError("articles", errors.New("invalid slug")))
		return
	}

	article := uaw.Article

	for _, tagM := range articleModel.Tags {
		article.Tags = append(article.Tags, tagM.Tag)
	}
	if err := a.ArticleRepo.SetTags(&articleModel, article.Tags); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Error binding or validating the request body", "error": err.Error()})
		return
	}

	articleModel.Slug = util.UserUpdateIfNotEmpty(articleModel.Slug, slug.Make(article.Title))
	articleModel.Title = util.UserUpdateIfNotEmpty(articleModel.Title, article.Title)
	articleModel.Description = util.UserUpdateIfNotEmpty(articleModel.Description, article.Description)
	articleModel.Body = util.UserUpdateIfNotEmpty(articleModel.Body, article.Body)

	if err := a.ArticleRepo.Update(&articleModel); err != nil {
		c.JSON(http.StatusUnprocessableEntity, util.NewError("database", err))
		return
	}
	loginUser := getLoginUser(c)
	loginUserArticle := a.ArticleRepo.GetArticleUserModel(loginUser)
	response := makeArticleResponse(a.ArticleRepo.GatherLoginUserStat(loginUserArticle, articleModel))
	c.JSON(http.StatusOK, gin.H{"article": response})
}

func (a *ArticleHandlerImpl) ArticleDelete(c *gin.Context) {
	slug := c.Param("slug")
	err := a.ArticleRepo.Delete(&repository.ArticleModel{Slug: slug})
	if err != nil {
		c.JSON(http.StatusNotFound, util.NewError("articles", errors.New("invalid slug")))
		return
	}
	c.JSON(http.StatusOK, gin.H{"article": "Delete success"})
}

func (a *ArticleHandlerImpl) ArticleFavorite(c *gin.Context) {
	slug := c.Param("slug")
	articleModel, err := a.ArticleRepo.FindOneArticle(&repository.ArticleModel{Slug: slug})
	if err != nil {
		c.JSON(http.StatusNotFound, util.NewError("articles", errors.New("invalid slug")))
		return
	}
	loginUser := getLoginUser(c)
	loginUserArticle := a.ArticleRepo.GetArticleUserModel(loginUser)
	if err = a.ArticleRepo.FavoriteBy(articleModel, loginUserArticle); err != nil {
		c.JSON(http.StatusInternalServerError, util.NewError("articles", errors.New("error when favorite")))
		return
	}
	response := makeArticleResponse(a.ArticleRepo.GatherLoginUserStat(loginUserArticle, articleModel))
	c.JSON(http.StatusOK, gin.H{"article": response})
}

func (a *ArticleHandlerImpl) ArticleUnfavorite(c *gin.Context) {
	slug := c.Param("slug")

	articleModel, err := a.ArticleRepo.FindOneArticle(&repository.ArticleModel{Slug: slug})
	if err != nil {
		c.JSON(http.StatusNotFound, util.NewError("articles", errors.New("invalid slug")))
		return
	}

	loginUser := getLoginUser(c)
	loginUserArticle := a.ArticleRepo.GetArticleUserModel(loginUser)
	if err = a.ArticleRepo.UnfavoriteBy(articleModel, loginUserArticle); err != nil {
		c.JSON(http.StatusInternalServerError, util.NewError("articles", errors.New("error when unfavorite")))
		return
	}
	response := makeArticleResponse(a.ArticleRepo.GatherLoginUserStat(loginUserArticle, articleModel))
	c.JSON(http.StatusOK, gin.H{"article": response})
}

func (a *ArticleHandlerImpl) ArticleCommentCreate(c *gin.Context) {
	var cw model.CommentWrapper
	if err := c.BindJSON(&cw); err != nil {
		c.JSON(http.StatusUnprocessableEntity, util.NewValidatorError(err))
		return
	}
	comment := cw.Comment

	slug := c.Param("slug")
	articleModel, err := a.ArticleRepo.FindOneArticle(&repository.ArticleModel{Slug: slug})
	if err != nil {
		c.JSON(http.StatusNotFound, util.NewError("comment", errors.New("invalid slug")))
		return
	}

	loginUser := getLoginUser(c)
	loginAum := a.ArticleRepo.GetArticleUserModel(loginUser)
	commentModel := repository.CommentModel{
		Article:   articleModel,
		ArticleID: articleModel.ID,
		Author:    loginAum,
		AuthorID:  loginAum.UserModelID,
		Body:      comment.Body,
	}

	if err := a.ArticleRepo.SaveOne(&commentModel); err != nil {
		c.JSON(http.StatusUnprocessableEntity, util.NewError("database", err))
		return
	}
	followingFlag, err := a.UsersRepo.IsFollowings(loginUser, articleModel.Author.UserModel)
	if err != nil {
		followingFlag = false // default (unknown) is not following
	}
	response := makeCommentResponse(commentModel, followingFlag)

	c.JSON(http.StatusCreated, gin.H{"comment": response})
}

func (a *ArticleHandlerImpl) ArticleCommentDelete(c *gin.Context) {
	id64, err := strconv.ParseUint(c.Param("id"), 10, 32)
	id := uint(id64)
	if err != nil {
		c.JSON(http.StatusNotFound, util.NewError("comment", errors.New("invalid id")))
		return
	}
	err = a.ArticleRepo.Delete(&repository.CommentModel{Model: gorm.Model{ID: id}})
	if err != nil {
		c.JSON(http.StatusNotFound, util.NewError("comment", errors.New("invalid id")))
		return
	}
	c.JSON(http.StatusOK, gin.H{"comment": "Delete success"})
}

func (a *ArticleHandlerImpl) ArticleCommentList(c *gin.Context) {
	slug := c.Param("slug")

	articleModel, err := a.ArticleRepo.FindOneArticle(&repository.ArticleModel{Slug: slug})
	if err != nil {
		c.JSON(http.StatusNotFound, util.NewError("comments", errors.New("invalid slug")))
		return
	}
	err = a.ArticleRepo.GetComments(&articleModel)
	if err != nil {
		c.JSON(http.StatusNotFound, util.NewError("comments", errors.New("database error")))
		return
	}

	loginUser := getLoginUser(c)
	response := []model.CommentResponse{}
	for _, comment := range articleModel.Comments {
		followingFlag, err := a.UsersRepo.IsFollowings(loginUser, comment.Author.UserModel)
		if err != nil {
			followingFlag = false // default (unknown) is not following
		}
		response = append(response, makeCommentResponse(comment, followingFlag))
	}

	c.JSON(http.StatusOK, gin.H{"comments": response})
}

func (a *ArticleHandlerImpl) TagList(c *gin.Context) {
	tagModels, err := a.ArticleRepo.GetAllTags()
	if err != nil {
		c.JSON(http.StatusNotFound, util.NewError("tags", errors.New("database error")))
		return
	}
	c.JSON(http.StatusOK, gin.H{"tags": makeTagsResponse(tagModels)})
}

// ----------------------------------------------------------

func makeArticleResponse(followingFlag, favorite bool, favoriteCount uint,
	am repository.ArticleModel) model.ArticleResponse {
	response := model.ArticleResponse{
		ID:             am.ID,
		Slug:           slug.Make(am.Title),
		Title:          am.Title,
		Description:    am.Description,
		Body:           am.Body,
		CreatedAt:      am.CreatedAt.UTC().Format("2006-01-02T15:04:05.999Z"),
		UpdatedAt:      am.UpdatedAt.UTC().Format("2006-01-02T15:04:05.999Z"),
		Author:         makeUserProfileResponse(am.Author.UserModel, followingFlag),
		Favorite:       favorite,
		FavoritesCount: favoriteCount,
	}
	response.Tags = makeTagsResponse(am.Tags)
	return response
}

func makeUserProfileResponse(authorUm repository.UserModel,
	followingFlag bool) model.ProfileResponse {
	pR := model.ProfileResponse{
		ID:        authorUm.ID,
		Username:  authorUm.Username,
		Bio:       authorUm.Bio,
		Image:     authorUm.Image,
		Following: followingFlag,
	}
	return pR
}

func makeTagResponse(tm repository.TagModel) string {
	return tm.Tag
}

func makeTagsResponse(tags []repository.TagModel) []string {
	response := []string{}
	for _, tag := range tags {
		response = append(response, makeTagResponse(tag))
	}
	return response
}

func makeCommentResponse(cm repository.CommentModel, followingFlag bool) model.CommentResponse {
	response := model.CommentResponse{
		ID:        cm.ID,
		Body:      cm.Body,
		CreatedAt: util.FormatTimestamp(cm.CreatedAt),
		UpdatedAt: util.FormatTimestamp(cm.UpdatedAt),
		Author:    makeUserProfileResponse(cm.Author.UserModel, followingFlag),
	}
	return response
}
