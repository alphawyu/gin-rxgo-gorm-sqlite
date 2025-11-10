package handler_test

import (
	"com/realworld/ginrxgogorm/handler"
	"com/realworld/ginrxgogorm/repository"
	mock_repository "com/realworld/ginrxgogorm/repository/mock"
	"com/realworld/ginrxgogorm/test_data"
	"errors"

	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gosimple/slug"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"gorm.io/gorm"
)

var _ = Describe("Testing End Points", func() {
	const (
		correlationId = "anyCorrelationId"
	)

	var (
		writer          *httptest.ResponseRecorder
		mockUserRepo    *mock_repository.MockUsersRepository
		mockArticleRepo *mock_repository.MockArticleRepository
		gCtx            *gin.Context
		testHandler     handler.ArticleHandler
		reqTemplate     = http.Request{
			Header: http.Header{
				"Content-Type":     []string{"application/json"},
				"X-Correlation-ID": []string{correlationId},
			},
			Method: http.MethodPost,
		}
		testArticle = repository.ArticleModel{
			Title:       "title1",
			Description: "desc1",
			Body:        "body1",
			Author: repository.ArticleUserModel{
				UserModel: repository.UserModel{
					Model: gorm.Model{ID: 12},
				},
			},
			Comments: []repository.CommentModel{
				{Author: repository.ArticleUserModel{
					UserModel: repository.UserModel{
						Model: gorm.Model{ID: 21},
					},
				}},
				{Author: repository.ArticleUserModel{
					UserModel: repository.UserModel{
						Model: gorm.Model{ID: 21},
					},
				}},
			},
			Tags: []repository.TagModel{
				{Tag: "tag1"},
				{Tag: "tag2"},
			},
		}
	)

	var ctrl *gomock.Controller
	BeforeEach(func() {
		// Skip("generic")
		ctrl = gomock.NewController(GinkgoT())
		mockUserRepo = mock_repository.NewMockUsersRepository(ctrl)
		mockArticleRepo = mock_repository.NewMockArticleRepository(ctrl)
		testHandler = handler.NewArticleHandler(mockUserRepo, mockArticleRepo)
		writer = httptest.NewRecorder()
		gCtx, _ = gin.CreateTestContext(writer)
		setLoginUser(gCtx, &repository.UserModel{Model: gorm.Model{ID: 99}})
	})
	AfterEach(func() {
		ctrl.Finish()
	})

	Context("/article", func() {
		Context("ArticleHandler.ArticleCreate()", func() {
			DescribeTable("testing...",
				func(requestPayload string, setupMock func(), oHttpStatus int, oResPayloadStr string) {
					req := reqTemplate
					req.Body = io.NopCloser(strings.NewReader(requestPayload))
					gCtx.Request = &req
					setupMock()

					testHandler.ArticleCreate(gCtx)

					Expect(writer.Body.String()).To(MatchJSON(oResPayloadStr))
					Expect(writer.Code).To(Equal(oHttpStatus))
				},
				Entry("should return StatusCreated, if create article successfully",
					`{
						"article": {
							"title": "title1",
							"description": "desc1",
							"body": "body1",
							"tagList": [
								"t1-1",
								"t1-2"
							]
						}
					}`,
					func() {
						mockArticleRepo.EXPECT().GetArticleUserModel(gomock.Cond(func(arg any) bool {
							val := arg.(repository.UserModel)
							return val.ID == 99
						})).Return(repository.ArticleUserModel{Model: gorm.Model{ID: 49}}).Times(1)
						mockArticleRepo.EXPECT().SaveOne(gomock.Cond(func(arg any) bool {
							val := arg.(*repository.ArticleModel)
							return val.Title == "title1" && val.Description == "desc1"
						})).Return(nil).Times(1)
						mockArticleRepo.EXPECT().GatherLoginUserStat(
							gomock.Cond(func(arg any) bool {
								val := arg.(repository.ArticleUserModel)
								return val.ID == 49
							}), gomock.Cond(func(arg any) bool {
								val := arg.(repository.ArticleModel)
								return val.Title == "title1" && val.Description == "desc1"
							})).Return(true, true, uint(5), testArticle).Times(1)
					},
					http.StatusCreated,
					`{
						"article": {
						"title": "title1",
						"slug": "title1",
						"description": "desc1",
						"body": "body1",
						"createdAt": "0001-01-01T00:00:00Z",
						"updatedAt": "0001-01-01T00:00:00Z",
						"author": {
							"following": true
						},
						"tagList": [
							"tag1",
							"tag2"
						],
						"favorited": true,
						"favoritesCount": 5
						}
					}`),
				Entry("should return StatusBadRequest, when multiple required fields are missing",
					`{  "article": {}}`,
					func() {},
					http.StatusBadRequest,
					`{"errors":{"Title":"{key: required}"}}`),
				Entry("should return StatusUnprocessableEntity, when saveOne failed",
					`{ "article": { "title": "title1" } }`,
					func() {
						mockArticleRepo.EXPECT().GetArticleUserModel(gomock.Any()).
							Return(repository.ArticleUserModel{Model: gorm.Model{ID: 49}}).Times(1)
						mockArticleRepo.EXPECT().SaveOne(gomock.Cond(func(arg any) bool {
							val := arg.(*repository.ArticleModel)
							return val.Title == "title1" && val.Description == ""
						})).Return(errors.New("some error")).Times(1)
					},
					http.StatusUnprocessableEntity,
					`{"errors":{"database":"some error"}}`),
			)
		})

		Context("ArticleHandler.ArticleUpdate()", func() {
			DescribeTable("testing...",
				func(requestPayload string, setupMock func(), oHttpStatus int, oResPayloadStr string) {
					req := reqTemplate
					req.Body = io.NopCloser(strings.NewReader(requestPayload))
					gCtx.Request = &req
					gCtx.Params = []gin.Param{
						{Key: "slug", Value: "s0"},
					}

					setupMock()

					testHandler.ArticleUpdate(gCtx)

					Expect(writer.Body.String()).To(MatchJSON(oResPayloadStr))
					Expect(writer.Code).To(Equal(oHttpStatus))
				},
				Entry("should return StatusOK, if update article successfully",
					`{
						"article": {
							"title": "title2",
							"description": "desc2",
							"body": "body1",
							"tagList": [
								"t1-1",
								"t1-2"
							]
						}
					}`,
					func() {
						mockArticleRepo.EXPECT().FindOneArticle(gomock.Cond(func(arg any) bool {
							val := arg.(*repository.ArticleModel)
							return val.Slug == slug.Make("s0")
						})).Return(testArticle, nil).Times(1)
						mockArticleRepo.EXPECT().GetArticleUserModel(gomock.Cond(func(arg any) bool {
							val := arg.(repository.UserModel)
							return val.ID == 99
						})).Return(repository.ArticleUserModel{Model: gorm.Model{ID: 49}}).Times(1)
						mockArticleRepo.EXPECT().SetTags(gomock.Cond(func(arg any) bool {
							val, _ := arg.(*repository.ArticleModel)
							return val.Title == "title1" && val.Description == "desc1"
						}), gomock.Any()).Return(nil).Times(1)
						mockArticleRepo.EXPECT().Update(gomock.Cond(func(arg any) bool {
							val := arg.(*repository.ArticleModel)
							return val.Title == "title2" && val.Description == "desc2"
						})).Return(nil).Times(1)
						mockArticleRepo.EXPECT().GatherLoginUserStat(
							gomock.Cond(func(arg any) bool {
								val := arg.(repository.ArticleUserModel)
								return val.ID == 49
							}), gomock.Cond(func(arg any) bool {
								val := arg.(repository.ArticleModel)
								return val.Title == "title2" && val.Description == "desc2"
							})).Return(true, true, uint(5), testArticle).Times(1)
					},
					http.StatusOK,
					`{
						"article": {
						"title": "title1",
						"slug": "title1",
						"description": "desc1",
						"body": "body1",
						"createdAt": "0001-01-01T00:00:00Z",
						"updatedAt": "0001-01-01T00:00:00Z",
						"author": {
							"following": true
						},
						"tagList": [
							"tag1",
							"tag2"
						],
						"favorited": true,
						"favoritesCount": 5
						}
					}`),
				Entry("should return StatusBadRequest, when request is not valid json",
					`{ "article": { "title": "title1" }`,
					func() {},
					http.StatusBadRequest,
					`{
						"errors": {
						"error": "unexpected EOF"
						}
					}`),
				Entry("should return StatusNotFound, when article cannot be found by slug",
					`{ "article": { "title": "title1" }}`,
					func() {
						mockArticleRepo.EXPECT().FindOneArticle(gomock.Any()).
							Return(testArticle, errors.New("some error")).Times(1)
					},
					http.StatusNotFound,
					`{
						"errors": {
						"articles": "invalid slug"
						}
					}`),
				Entry("should return StatusBadRequest, when setTags call failed",
					`{ "article": { "title": "title1" } }`,
					func() {
						mockArticleRepo.EXPECT().FindOneArticle(gomock.Cond(func(arg any) bool {
							val := arg.(*repository.ArticleModel)
							return val.Slug == slug.Make("s0")
						})).Return(testArticle, nil).Times(1)
						mockArticleRepo.EXPECT().SetTags(gomock.Cond(func(arg any) bool {
							val, _ := arg.(*repository.ArticleModel)
							return val.Title == "title1" && val.Description == "desc1"
						}), gomock.Any()).Return(errors.New("some error")).Times(1)
					},
					http.StatusBadRequest,
					`{"error":"some error","message":"Error binding or validating the request body"}`),
				Entry("should return StatusUnprocessableEntity, when update article failed",
					`{ "article": { "title": "title2" } }`,
					func() {
						mockArticleRepo.EXPECT().FindOneArticle(gomock.Cond(func(arg any) bool {
							val := arg.(*repository.ArticleModel)
							return val.Slug == slug.Make("s0")
						})).Return(testArticle, nil).Times(1)
						mockArticleRepo.EXPECT().SetTags(gomock.Cond(func(arg any) bool {
							val, _ := arg.(*repository.ArticleModel)
							return val.Title == "title1" && val.Description == "desc1"
						}), gomock.Any()).Return(nil).Times(1)
						mockArticleRepo.EXPECT().Update(gomock.Cond(func(arg any) bool {
							val := arg.(*repository.ArticleModel)
							return val.Title == "title2" && val.Description == "desc1"
						})).Return(errors.New("some error")).Times(1)
					},
					http.StatusUnprocessableEntity,
					`{"errors":{"database":"some error"}}`),
			)
		})

		Context("ArticleHandler.ArticleDelete()", func() {
			DescribeTable("testing...",
				func(setupMock func(), oHttpStatus int, oResPayloadStr string) {
					gCtx.AddParam("slug", "s0")

					setupMock()

					testHandler.ArticleDelete(gCtx)

					Expect(writer.Body.String()).To(MatchJSON(oResPayloadStr))
					Expect(writer.Code).To(Equal(oHttpStatus))
				},
				Entry("should return StatusOK, if delete article successfully",
					func() {
						mockArticleRepo.EXPECT().DeleteArticle(gomock.Cond(func(arg any) bool {
							val := arg.(*repository.ArticleModel)
							return val.Slug == slug.Make("s0")
						})).Return(nil).Times(1)
					},
					http.StatusOK,
					`{
						"article": "Delete success"
					}`),
				Entry("should return StatusNotFound, when article cannot be found by slug",
					func() {
						mockArticleRepo.EXPECT().DeleteArticle(gomock.Any()).
							Return(errors.New("some error")).Times(1)
					},
					http.StatusNotFound,
					`{
						"errors": {
						"articles": "invalid slug"
						}
					}`),
			)
		})

		Context("ArticleHandler.ArticleList()", func() {
			DescribeTable("testing...",
				func(setupMock func(), oHttpStatus int, oResPayloadStr string) {
					reqURL, _ := url.Parse("http://localhost:8080/test?tag=t0&author=a0&favorited=f0&limit=99&offset=11")
					gCtx.Request = &http.Request{
						URL:    reqURL,
						Header: make(http.Header),
					}

					setupMock()

					testHandler.ArticleList(gCtx)

					Expect(writer.Body.String()).To(MatchJSON(oResPayloadStr))
					Expect(writer.Code).To(Equal(oHttpStatus))
				},
				Entry("should return StatusOK, if find articles successfully",
					func() {
						mockArticleRepo.EXPECT().FindManyArticle("t0", "a0", "99", "11", "f0").
							Return(test_data.GenerateTestArticles(10, 2, true, nil), 2, nil).Times(1)
						mockArticleRepo.EXPECT().GetArticleUserModel(gomock.Cond(func(arg any) bool {
							val := arg.(repository.UserModel)
							return val.ID == 99
						})).Return(repository.ArticleUserModel{}).Times(1)
						mockArticleRepo.EXPECT().GatherLoginUserStat(gomock.Any(), gomock.Any()).
							Return(true, true, uint(5), testArticle).Times(2)
					},
					http.StatusOK,
					`{
						"articles": [
						{
							"title": "title1",
							"slug": "title1",
							"description": "desc1",
							"body": "body1",
							"createdAt": "0001-01-01T00:00:00Z",
							"updatedAt": "0001-01-01T00:00:00Z",
							"author": {
							"following": true
							},
							"tagList": [
							"tag1",
							"tag2"
							],
							"favorited": true,
							"favoritesCount": 5
						},
						{
							"title": "title1",
							"slug": "title1",
							"description": "desc1",
							"body": "body1",
							"createdAt": "0001-01-01T00:00:00Z",
							"updatedAt": "0001-01-01T00:00:00Z",
							"author": {
							"following": true
							},
							"tagList": [
							"tag1",
							"tag2"
							],
							"favorited": true,
							"favoritesCount": 5
						}
						],
						"articlesCount": 2
					}`),
				Entry("should return StatusOK with empty response, when no article is found",
					func() {
						mockArticleRepo.EXPECT().FindManyArticle("t0", "a0", "99", "11", "f0").
							Return([]repository.ArticleModel{}, 0, nil).Times(1)
						mockArticleRepo.EXPECT().GetArticleUserModel(gomock.Any()).
							Return(repository.ArticleUserModel{Model: gorm.Model{ID: 49}}).Times(1)
					},
					http.StatusOK,
					`{
						"articles": [],
						"articlesCount": 0
					}`),
				Entry("should return StatusOK, when error no article look up",
					func() {
						mockArticleRepo.EXPECT().FindManyArticle("t0", "a0", "99", "11", "f0").
							Return([]repository.ArticleModel{}, 0, errors.New("some error")).Times(1)
					},
					http.StatusNotFound,
					`{
						"errors": {
						"articles": "invalid param"
						}
					}`),
			)
		})

		Context("ArticleHandler.ArticleRetrieve()", func() {
			DescribeTable("testing...",
				func(setupMock func(), oHttpStatus int, oResPayloadStr string) {
					gCtx.AddParam("slug", "s0")

					setupMock()

					testHandler.ArticleRetrieve(gCtx)

					Expect(writer.Body.String()).To(MatchJSON(oResPayloadStr))
					Expect(writer.Code).To(Equal(oHttpStatus))
				},
				Entry("should return StatusOK, if retrieve article successfully",
					func() {
						mockArticleRepo.EXPECT().FindOneArticle(gomock.Cond(func(arg any) bool {
							val := arg.(*repository.ArticleModel)
							return val.Slug == slug.Make("s0")
						})).Return(testArticle, nil).Times(1)
						mockArticleRepo.EXPECT().GetArticleUserModel(gomock.Cond(func(arg any) bool {
							val := arg.(repository.UserModel)
							return val.ID == 99
						})).Return(repository.ArticleUserModel{Model: gorm.Model{ID: 49}}).Times(1)
						mockArticleRepo.EXPECT().GatherLoginUserStat(
							gomock.Cond(func(arg any) bool {
								val := arg.(repository.ArticleUserModel)
								return val.ID == 49
							}), gomock.Cond(func(arg any) bool {
								val := arg.(repository.ArticleModel)
								return val.Title == "title1" && val.Description == "desc1"
							})).Return(true, true, uint(5), testArticle).Times(1)
					},
					http.StatusOK,
					`{
						"article": {
						"title": "title1",
						"slug": "title1",
						"description": "desc1",
						"body": "body1",
						"createdAt": "0001-01-01T00:00:00Z",
						"updatedAt": "0001-01-01T00:00:00Z",
						"author": {
							"following": true
						},
						"tagList": [
							"tag1",
							"tag2"
						],
						"favorited": true,
						"favoritesCount": 5
						}
					}`),
				Entry("should return StatusNotFound, when article cannot be found by slug",
					func() {
						mockArticleRepo.EXPECT().FindOneArticle(gomock.Any()).
							Return(testArticle, errors.New("some error")).Times(1)
					},
					http.StatusNotFound,
					`{
						"errors": {
						"articles": "invalid slug"
						}
					}`),
			)
		})

		Context("ArticleHandler.ArticleFeed()", func() {
			DescribeTable("testing...",
				func(setupMock func(), oHttpStatus int, oResPayloadStr string) {
					reqURL, _ := url.Parse("http://localhost:8080/test?limit=99&offset=11")
					gCtx.Request = &http.Request{
						URL:    reqURL,
						Header: make(http.Header),
					}

					setupMock()

					testHandler.ArticleFeed(gCtx)

					Expect(writer.Body.String()).To(MatchJSON(oResPayloadStr))
					Expect(writer.Code).To(Equal(oHttpStatus))
				},
				Entry("should return StatusOK, if article feed successfully",
					func() {
						mockArticleRepo.EXPECT().GetArticleUserModel(gomock.Cond(func(arg any) bool {
							val := arg.(repository.UserModel)
							return val.ID == 99
						})).Return(repository.ArticleUserModel{Model: gorm.Model{ID: 49}}).Times(1)
						mockArticleRepo.EXPECT().GetArticleFeed(gomock.Any(), "99", "11").
							Return(test_data.GenerateTestArticles(10, 2, true, nil), 2, nil).Times(1)
						mockArticleRepo.EXPECT().GatherLoginUserStat(gomock.Any(), gomock.Any()).
							Return(true, true, uint(5), testArticle).Times(2)
					},
					http.StatusOK,
					`{
						"articles": [
						{
							"title": "title1",
							"slug": "title1",
							"description": "desc1",
							"body": "body1",
							"createdAt": "0001-01-01T00:00:00Z",
							"updatedAt": "0001-01-01T00:00:00Z",
							"author": {
							"following": true
							},
							"tagList": [
							"tag1",
							"tag2"
							],
							"favorited": true,
							"favoritesCount": 5
						},
						{
							"title": "title1",
							"slug": "title1",
							"description": "desc1",
							"body": "body1",
							"createdAt": "0001-01-01T00:00:00Z",
							"updatedAt": "0001-01-01T00:00:00Z",
							"author": {
							"following": true
							},
							"tagList": [
							"tag1",
							"tag2"
							],
							"favorited": true,
							"favoritesCount": 5
						}
						],
						"articlesCount": 2
					}`),
				Entry("should return StatusOK with empty response, when no article is found",
					func() {
						mockArticleRepo.EXPECT().GetArticleUserModel(gomock.Any()).
							Return(repository.ArticleUserModel{Model: gorm.Model{ID: 49}}).Times(1)
						mockArticleRepo.EXPECT().GetArticleFeed(gomock.Any(), "99", "11").
							Return([]repository.ArticleModel{}, 0, nil).Times(1)
					},
					http.StatusOK,
					`{
						"articles": [],
						"articlesCount": 0
					}`),
				Entry("should return StatusOK, when error no article look up",
					func() {
						mockArticleRepo.EXPECT().GetArticleUserModel(gomock.Any()).
							Return(repository.ArticleUserModel{Model: gorm.Model{ID: 49}}).Times(1)
						mockArticleRepo.EXPECT().GetArticleFeed(gomock.Any(), "99", "11").
							Return([]repository.ArticleModel{}, 0, errors.New("some error")).Times(1)
					},
					http.StatusNotFound,
					`{
						"errors": {
						"articles": "invalid param"
						}
					}`),
			)

			It("should return StatusNotFound, when login user is invalid", func() {
				reqURL, _ := url.Parse("http://localhost:8080/test?limit=99&offset=11")
				gCtx.Request = &http.Request{
					URL:    reqURL,
					Header: make(http.Header),
				}

				setLoginUser(gCtx, &repository.UserModel{Model: gorm.Model{ID: 0}})

				testHandler.ArticleFeed(gCtx)

				Expect(writer.Body.String()).To(Equal(`{"errors":{"articles":"Require Current Login User!"}}`))
				Expect(writer.Code).To(Equal(http.StatusNotFound))
			})
		})

		Context("ArticleHandler.ArticleFavorite()", func() {
			DescribeTable("testing...",
				func(setupMock func(), oHttpStatus int, oResPayloadStr string) {
					gCtx.AddParam("slug", "s0")

					setupMock()

					testHandler.ArticleFavorite(gCtx)

					Expect(writer.Body.String()).To(MatchJSON(oResPayloadStr))
					Expect(writer.Code).To(Equal(oHttpStatus))
				},
				Entry("should return StatusOK, if favorite the article successfully",
					func() {
						mockArticleRepo.EXPECT().FindOneArticle(gomock.Cond(func(arg any) bool {
							val := arg.(*repository.ArticleModel)
							return val.Slug == slug.Make("s0")
						})).Return(testArticle, nil).Times(1)
						mockArticleRepo.EXPECT().GetArticleUserModel(gomock.Cond(func(arg any) bool {
							val := arg.(repository.UserModel)
							return val.ID == 99
						})).Return(repository.ArticleUserModel{Model: gorm.Model{ID: 49}}).Times(1)
						mockArticleRepo.EXPECT().FavoriteBy(gomock.Any(), gomock.Any()).
							Return(nil).Times(1)
						mockArticleRepo.EXPECT().GatherLoginUserStat(
							gomock.Cond(func(arg any) bool {
								val := arg.(repository.ArticleUserModel)
								return val.ID == 49
							}), gomock.Cond(func(arg any) bool {
								val := arg.(repository.ArticleModel)
								return val.Title == "title1" && val.Description == "desc1"
							})).Return(true, true, uint(5), testArticle).Times(1)
					},
					http.StatusOK,
					`{
						"article": {
						"title": "title1",
						"slug": "title1",
						"description": "desc1",
						"body": "body1",
						"createdAt": "0001-01-01T00:00:00Z",
						"updatedAt": "0001-01-01T00:00:00Z",
						"author": {
							"following": true
						},
						"tagList": [
							"tag1",
							"tag2"
						],
						"favorited": true,
						"favoritesCount": 5
						}
					}`),
				Entry("should return StatusNotFound, when article cannot be found by slug",
					func() {
						mockArticleRepo.EXPECT().FindOneArticle(gomock.Any()).
							Return(testArticle, errors.New("some error")).Times(1)
					},
					http.StatusNotFound,
					`{
						"errors": {
						"articles": "invalid slug"
						}
					}`),
				Entry("should return StatusInternalServerError, when error on favorite the article",
					func() {
						mockArticleRepo.EXPECT().FindOneArticle(gomock.Any()).
							Return(testArticle, nil).Times(1)
						mockArticleRepo.EXPECT().GetArticleUserModel(gomock.Any()).
							Return(repository.ArticleUserModel{Model: gorm.Model{ID: 49}}).Times(1)
						mockArticleRepo.EXPECT().FavoriteBy(gomock.Any(), gomock.Any()).
							Return(errors.New("some error")).Times(1)
					},
					http.StatusInternalServerError,
					`{
						"errors": {
						"articles": "error when favorite"
						}
					}`),
			)
		})

		Context("ArticleHandler.ArticleUnfavorite()", func() {
			DescribeTable("testing...",
				func(setupMock func(), oHttpStatus int, oResPayloadStr string) {
					gCtx.AddParam("slug", "s0")

					setupMock()

					testHandler.ArticleUnfavorite(gCtx)

					Expect(writer.Body.String()).To(MatchJSON(oResPayloadStr))
					Expect(writer.Code).To(Equal(oHttpStatus))
				},
				Entry("should return StatusOK, if unfavorite the article successfully",
					func() {
						mockArticleRepo.EXPECT().FindOneArticle(gomock.Cond(func(arg any) bool {
							val := arg.(*repository.ArticleModel)
							return val.Slug == slug.Make("s0")
						})).Return(testArticle, nil).Times(1)
						mockArticleRepo.EXPECT().GetArticleUserModel(gomock.Cond(func(arg any) bool {
							val := arg.(repository.UserModel)
							return val.ID == 99
						})).Return(repository.ArticleUserModel{Model: gorm.Model{ID: 49}}).Times(1)
						mockArticleRepo.EXPECT().UnfavoriteBy(gomock.Any(), gomock.Any()).
							Return(nil).Times(1)
						mockArticleRepo.EXPECT().GatherLoginUserStat(
							gomock.Cond(func(arg any) bool {
								val := arg.(repository.ArticleUserModel)
								return val.ID == 49
							}), gomock.Cond(func(arg any) bool {
								val := arg.(repository.ArticleModel)
								return val.Title == "title1" && val.Description == "desc1"
							})).Return(true, true, uint(5), testArticle).Times(1)
					},
					http.StatusOK,
					`{
						"article": {
						"title": "title1",
						"slug": "title1",
						"description": "desc1",
						"body": "body1",
						"createdAt": "0001-01-01T00:00:00Z",
						"updatedAt": "0001-01-01T00:00:00Z",
						"author": {
							"following": true
						},
						"tagList": [
							"tag1",
							"tag2"
						],
						"favorited": true,
						"favoritesCount": 5
						}
					}`),
				Entry("should return StatusNotFound, when article cannot be found by slug",
					func() {
						mockArticleRepo.EXPECT().FindOneArticle(gomock.Any()).
							Return(testArticle, errors.New("some error")).Times(1)
					},
					http.StatusNotFound,
					`{
						"errors": {
						"articles": "invalid slug"
						}
					}`),
				Entry("should return StatusInternalServerError, when error on unfavorite the article",
					func() {
						mockArticleRepo.EXPECT().FindOneArticle(gomock.Any()).
							Return(testArticle, nil).Times(1)
						mockArticleRepo.EXPECT().GetArticleUserModel(gomock.Any()).
							Return(repository.ArticleUserModel{Model: gorm.Model{ID: 49}}).Times(1)
						mockArticleRepo.EXPECT().UnfavoriteBy(gomock.Any(), gomock.Any()).
							Return(errors.New("some error")).Times(1)
					},
					http.StatusInternalServerError,
					`{ "errors": { "articles": "error when unfavorite" } }`),
			)
		})

		// ArticleCommentCreate(c *gin.Context)
		Context("ArticleHandler.ArticleCommentCreate()", func() {
			DescribeTable("testing...",
				func(requestPayload string, setupMock func(), oHttpStatus int, oResPayloadStr string) {
					req := reqTemplate
					req.Body = io.NopCloser(strings.NewReader(requestPayload))
					gCtx.Request = &req
					gCtx.AddParam("slug", "s0")

					setupMock()

					testHandler.ArticleCommentCreate(gCtx)

					Expect(writer.Body.String()).To(MatchJSON(oResPayloadStr))
					Expect(writer.Code).To(Equal(oHttpStatus))
				},
				Entry("should return StatusCreated, if comment is created successfully",
					`{"comment":{"body":"cb0"}}`,
					func() {
						mockArticleRepo.EXPECT().FindOneArticle(gomock.Cond(func(arg any) bool {
							val := arg.(*repository.ArticleModel)
							return val.Slug == slug.Make("s0")
						})).Return(testArticle, nil).Times(1)
						mockArticleRepo.EXPECT().GetArticleUserModel(gomock.Cond(func(arg any) bool {
							val := arg.(repository.UserModel)
							return val.ID == 99
						})).Return(repository.ArticleUserModel{Model: gorm.Model{ID: 49}}).Times(1)
						mockArticleRepo.EXPECT().SaveOne(gomock.Cond(func(arg any) bool {
							val := arg.(*repository.CommentModel)
							return val.Body == "cb0"
						})).Return(nil).Times(1)
						mockUserRepo.EXPECT().IsFollowings(gomock.Cond(func(arg any) bool {
							val := arg.(repository.UserModel)
							return val.ID == 99
						}), gomock.Cond(func(arg any) bool {
							val := arg.(repository.UserModel)
							return val.ID == 12
						})).Return(true, nil).Times(1)
					},
					http.StatusCreated,
					`{
						"comment": {
							"body": "cb0",
							"author": {
								"following": true
							}
						}
					}`),
				Entry("should return StatusBadRequest, when required body is missing",
					`{"comment":{}}`,
					func() {},
					http.StatusBadRequest,
					`{
						"errors": {
							"Body": "{key: required}"
						}
					}`),
				Entry("should return StatusNotFound, when article cannot be found by slug",
					`{"comment":{"body":"cb0"}}`,
					func() {
						mockArticleRepo.EXPECT().FindOneArticle(gomock.Any()).
							Return(testArticle, errors.New("some error")).Times(1)
					},
					http.StatusNotFound,
					`{
						"errors": {
							"comment": "invalid slug"
						}
					}`),
				Entry("should return StatusNotFound, when error on saving comment",
					`{"comment":{"body":"cb0"}}`,
					func() {
						mockArticleRepo.EXPECT().FindOneArticle(gomock.Any()).
							Return(testArticle, nil).Times(1)
						mockArticleRepo.EXPECT().GetArticleUserModel(gomock.Any()).
							Return(repository.ArticleUserModel{Model: gorm.Model{ID: 49}}).Times(1)
						mockArticleRepo.EXPECT().SaveOne(gomock.Any()).
							Return(errors.New("some error")).Times(1)
					},
					http.StatusUnprocessableEntity,
					`{ "errors": { "database": "some error" } }`),
				Entry("should return StatusCreated, with followings set to false",
					`{"comment":{"body":"cb0"}}`,
					func() {
						mockArticleRepo.EXPECT().FindOneArticle(gomock.Any()).
							Return(testArticle, nil).Times(1)
						mockArticleRepo.EXPECT().GetArticleUserModel(gomock.Any()).
							Return(repository.ArticleUserModel{Model: gorm.Model{ID: 49}}).Times(1)
						mockArticleRepo.EXPECT().SaveOne(gomock.Any()).
							Return(nil).Times(1)
						mockUserRepo.EXPECT().IsFollowings(gomock.Any(), gomock.Any()).
							Return(true, errors.New("some error")).Times(1)
					},
					http.StatusCreated,
					`{
						"comment": {
							"body": "cb0",
							"author": {
								"following": false
							}
					    }
					}`),
			)
		})

		Context("ArticleHandler.ArticleCommentDelete()", func() {
			DescribeTable("testing...",
				func(setupMock func(), oHttpStatus int, oResPayloadStr string) {
					setupMock()

					testHandler.ArticleCommentDelete(gCtx)

					Expect(writer.Body.String()).To(MatchJSON(oResPayloadStr))
					Expect(writer.Code).To(Equal(oHttpStatus))
				},
				Entry("should return StatusOK, if delete comment successfully",
					func() {
						gCtx.AddParam("id", "12")
						mockArticleRepo.EXPECT().DeleteComment(gomock.Cond(func(arg any) bool {
							val := arg.([]uint)
							return val[0] == uint(12)
						})).Return(nil).Times(1)
					},
					http.StatusOK,
					`{
						"comment": "Delete success"
					}`),
				Entry("should return StatusNotFound, when provided id is not a number",
					func() {
						gCtx.AddParam("id", "x12")
					},
					http.StatusNotFound,
					`{ "errors": { "comment": "invalid id" } }`),
				Entry("should return StatusNotFound, when cannot find comment by id",
					func() {
						gCtx.AddParam("id", "12")
						mockArticleRepo.EXPECT().DeleteComment(gomock.Any()).
							Return(errors.New("some error")).Times(1)
					},
					http.StatusNotFound,
					`{ "errors": { "comment": "invalid id" } }`),
			)
		})

		// ArticleCommentList(c *gin.Context)
		Context("ArticleHandler.ArticleCommentList()", func() {
			DescribeTable("testing...",
				func(setupMock func(), oHttpStatus int, oResPayloadStr string) {
					gCtx.AddParam("slug", "s0")

					setupMock()

					testHandler.ArticleCommentList(gCtx)

					Expect(writer.Body.String()).To(MatchJSON(oResPayloadStr))
					Expect(writer.Code).To(Equal(oHttpStatus))
				},
				Entry("should return StatusOK, if list comments successfully",
					func() {
						mockArticleRepo.EXPECT().FindOneArticle(gomock.Cond(func(arg any) bool {
							val := arg.(*repository.ArticleModel)
							return val.Slug == slug.Make("s0")
						})).Return(testArticle, nil).Times(1)
						mockArticleRepo.EXPECT().GetComments(gomock.Cond(func(arg any) bool {
							val := arg.(*repository.ArticleModel)
							return val.Title == "title1" && val.Description == "desc1"
						})).Return(nil).Times(1)
						mockUserRepo.EXPECT().IsFollowings(gomock.Cond(func(arg any) bool {
							val := arg.(repository.UserModel)
							return val.ID == 99
						}), gomock.Cond(func(arg any) bool {
							val := arg.(repository.UserModel)
							return val.ID == 21
						})).Return(true, nil).Times(2)
					},
					http.StatusOK,
					`{
						"comments": [
							{
								"author": {
								"following": true
								}
							},
							{
								"author": {
								"following": true
								}
							}
						]
					}`),
				Entry("should return StatusNotFound, when cannot find the by slug",
					func() {
						mockArticleRepo.EXPECT().FindOneArticle(gomock.Any()).
							Return(testArticle, errors.New("some error")).Times(1)
					},
					http.StatusNotFound,
					`{ "errors": { "comments": "invalid slug" } }`),
				Entry("should return StatusNotFound, when error on finding comments of the article",
					func() {
						mockArticleRepo.EXPECT().FindOneArticle(gomock.Any()).
							Return(testArticle, nil).Times(1)
						mockArticleRepo.EXPECT().GetComments(gomock.Any()).
							Return(errors.New("some error")).Times(1)

					},
					http.StatusNotFound,
					`{ "errors": { "comments": "database error" } }`),
				Entry("should return StatusOK, when error on finding followings",
					func() {
						mockArticleRepo.EXPECT().FindOneArticle(gomock.Any()).
							Return(testArticle, nil).Times(1)
						mockArticleRepo.EXPECT().GetComments(gomock.Any()).
							Return(nil).Times(1)
						mockUserRepo.EXPECT().IsFollowings(gomock.Any(), gomock.Any()).
							Return(true, errors.New("some error")).Times(2)
					},
					http.StatusOK,
					`{
						"comments": [
							{
								"author": {
								"following": false
								}
							},
							{
								"author": {
								"following": false
								}
							}
						]
					}`),
			)
		})

		// TagList(c *gin.Context)
		Context("ArticleHandler.TagList()", func() {
			DescribeTable("testing...",
				func(setupMock func(), oHttpStatus int, oResPayloadStr string) {
					setupMock()

					testHandler.TagList(gCtx)

					Expect(writer.Body.String()).To(MatchJSON(oResPayloadStr))
					Expect(writer.Code).To(Equal(oHttpStatus))
				},
				Entry("should return StatusOK, if article is deleted successfully",
					func() {
						mockArticleRepo.EXPECT().GetAllTags().
							Return([]repository.TagModel{
								{Tag: "tag1"},
								{Tag: "tag2"},
							}, nil).Times(1)
					},
					http.StatusOK,
					`{
						"tags": [
							"tag1",
							"tag2"
						]
					}`),
				Entry("should return StatusNotFound, when article cannot be found by slug",
					func() {
						mockArticleRepo.EXPECT().GetAllTags().
							Return([]repository.TagModel{}, errors.New("some error")).Times(1)
					},
					http.StatusNotFound,
					`{ "errors": { "tags": "database error" } }`),
			)
		})

	})
})
