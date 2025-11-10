package app_test

import (
	"com/realworld/ginrxgogorm/app"
	mock_handler "com/realworld/ginrxgogorm/handler/mock"
	"com/realworld/ginrxgogorm/middleware"
	mock_repository "com/realworld/ginrxgogorm/repository/mock"
	"com/realworld/ginrxgogorm/test_data"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Testing Router", Ordered, func() {
	const (
		correlationId        = "anyCorrelationId"
		AUTHORIZATION_HEADER = "Authorization"
	)

	var (
		router             *gin.Engine
		writer             *httptest.ResponseRecorder
		req                *http.Request
		err                error
		mockUserRepo       *mock_repository.MockUsersRepository
		mockUsersHandler   *mock_handler.MockUsersHandler
		mockArticleHandler *mock_handler.MockArticleHandler
		testApi            *app.App
		mockLoginUser      = test_data.GenerateTestUsers(9, 1, true)[0]
		bearerToken        = "Token " + middleware.GenToken(mockLoginUser.ID)
	)

	var ctrl *gomock.Controller
	BeforeEach(func() {
		// Skip("generic")
		ctrl = gomock.NewController(GinkgoT())
		mockUserRepo = mock_repository.NewMockUsersRepository(ctrl)
		mockUsersHandler = mock_handler.NewMockUsersHandler(ctrl)
		mockArticleHandler = mock_handler.NewMockArticleHandler(ctrl)
		testApi = &app.App{
			UsersRepo:      mockUserRepo,
			UsersHandler:   mockUsersHandler,
			ArticleHandler: mockArticleHandler,
		}
		writer = httptest.NewRecorder()
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("GET /health ", func() {
		router = testApi.SetupRouter()

		req, err = http.NewRequest(http.MethodGet, "/health", nil)
		Expect(err).To(BeNil())

		router.ServeHTTP(writer, req)

		Expect(writer.Body.String()).To(Equal(`{"alive":true}`))
		Expect(writer.Code).To(Equal(http.StatusOK))
	})

	Context("/api/users", func() {
		It("POST _ >> UsersHandler.UsersRegistration", func() {
			mockUsersHandler.EXPECT().UsersRegistration(gomock.Any()).Times(1).
				Do(func(c *gin.Context) {
					c.String(http.StatusOK, "UsersRegistration-OK")
				})

			router = testApi.SetupRouter()

			req, err = http.NewRequest(http.MethodPost, "/api/users", strings.NewReader("requestPayload"))
			Expect(err).To(BeNil())

			router.ServeHTTP(writer, req)

			Expect(writer.Body.String()).To(Equal("UsersRegistration-OK"))
			Expect(writer.Code).To(Equal(http.StatusOK))
		})

		It("POST /login >> UsersHandler.UsersLogin", func() {
			mockUsersHandler.EXPECT().UsersLogin(gomock.Any()).Times(1).
				Do(func(c *gin.Context) {
					c.String(http.StatusOK, "UsersLogin-OK")
				})

			router = testApi.SetupRouter()

			req, err = http.NewRequest(http.MethodPost, "/api/users/login", strings.NewReader("requestPayload"))
			Expect(err).To(BeNil())

			router.ServeHTTP(writer, req)

			Expect(writer.Body.String()).To(Equal("UsersLogin-OK"))
			Expect(writer.Code).To(Equal(http.StatusOK))
		})
	})

	Context("with AuthMiddleware, but allow anonymous access", func() {
		Context("/api/articles - allow anonymous access", func() {
			DescribeTable("GET _?tag=&author=&favorited=&limit=&offset= >> ArticleHandler.ArticleList",
				func(iUrl, oExpected string) {
					mockArticleHandler.EXPECT().ArticleList(gomock.Any()).Times(1).
						Do(func(c *gin.Context) {
							tag := c.Query("tag")
							author := c.Query("author")
							favorited := c.Query("favorited")
							limit := c.Query("limit")
							offset := c.Query("offset")
							c.String(http.StatusOK, "ArticleList-OK-tag=%s-author=%s-favorited=%s-limit=%s-offset=%s", tag, author, favorited, limit, offset)
						})

					router = testApi.SetupRouter()

					req, err = http.NewRequest(http.MethodGet, iUrl, nil)
					Expect(err).To(BeNil())

					router.ServeHTTP(writer, req)

					Expect(writer.Body.String()).To(Equal(oExpected))
					Expect(writer.Code).To(Equal(http.StatusOK))
				},
				Entry("no param", "/api/articles", "ArticleList-OK-tag=-author=-favorited=-limit=-offset="),
				Entry("with tag", "/api/articles?tag=t0", "ArticleList-OK-tag=t0-author=-favorited=-limit=-offset="),
				Entry("with author", "/api/articles?author=a0", "ArticleList-OK-tag=-author=a0-favorited=-limit=-offset="),
				Entry("with favorited", "/api/articles?favorited=f0", "ArticleList-OK-tag=-author=-favorited=f0-limit=-offset="),
				Entry("with limit", "/api/articles?limit=99", "ArticleList-OK-tag=-author=-favorited=-limit=99-offset="),
				Entry("with offset", "/api/articles?offset=11", "ArticleList-OK-tag=-author=-favorited=-limit=-offset=11"),
				Entry("with limit & offset", "/api/articles?limit=99&offset=11", "ArticleList-OK-tag=-author=-favorited=-limit=99-offset=11"),
				Entry("with offset & limit", "/api/articles?offset=11&limit=99&", "ArticleList-OK-tag=-author=-favorited=-limit=99-offset=11"),
			)

			DescribeTable("GET /feed?limit=&offset= >> ArticleHandler.ArticleFeed",
				func(iUrl, oExpected string) {
					mockArticleHandler.EXPECT().ArticleFeed(gomock.Any()).Times(1).
						Do(func(c *gin.Context) {
							limit := c.Query("limit")
							offset := c.Query("offset")
							c.String(http.StatusOK, "ArticleFeed-OK-limit=%s-offset=%s", limit, offset)
						})

					router = testApi.SetupRouter()

					req, err = http.NewRequest(http.MethodGet, iUrl, nil)
					Expect(err).To(BeNil())

					router.ServeHTTP(writer, req)

					Expect(writer.Body.String()).To(Equal(oExpected))
					Expect(writer.Code).To(Equal(http.StatusOK))
				},
				Entry("no param", "/api/articles/feed", "ArticleFeed-OK-limit=-offset="),
				Entry("with limit", "/api/articles/feed?limit=99", "ArticleFeed-OK-limit=99-offset="),
				Entry("with offset", "/api/articles/feed?offset=11", "ArticleFeed-OK-limit=-offset=11"),
				Entry("with limit & offset", "/api/articles/feed?limit=99&offset=11", "ArticleFeed-OK-limit=99-offset=11"),
				Entry("with offset & limit", "/api/articles/feed?offset=11&limit=99&", "ArticleFeed-OK-limit=99-offset=11"),
			)

			DescribeTable("GET /:slug >> ArticleHandler.ArticleRetrieve",
				func(iUrl, oExpected string) {
					mockArticleHandler.EXPECT().ArticleRetrieve(gomock.Any()).Times(1).
						Do(func(c *gin.Context) {
							slug := c.Param("slug")
							c.String(http.StatusOK, fmt.Sprintf("ArticleRetrieve-OK-slug=%s", slug))
						})

					router = testApi.SetupRouter()

					req, err = http.NewRequest(http.MethodGet, iUrl, nil)
					Expect(err).To(BeNil())

					router.ServeHTTP(writer, req)

					Expect(writer.Body.String()).To(Equal(oExpected))
					Expect(writer.Code).To(Equal(http.StatusOK))
				},
				// NOTE: empty slug at end does not work well, try uncomment next line and test for yourself
				// Entry("empty slug", "/api/articles/", "ArticleRetrieve-OK-slug="),
				Entry("normal slug", "/api/articles/s0", "ArticleRetrieve-OK-slug=s0"),
			)

			DescribeTable("GET /:slug/comments >> ArticleHandler.ArticleCommentList",
				func(iUrl, oExpected string) {
					mockArticleHandler.EXPECT().ArticleCommentList(gomock.Any()).Times(1).
						Do(func(c *gin.Context) {
							slug := c.Param("slug")
							c.String(http.StatusOK, fmt.Sprintf("ArticleCommentList-OK-slug=%s", slug))
						})

					router = testApi.SetupRouter()

					req, err = http.NewRequest(http.MethodGet, iUrl, nil)
					Expect(err).To(BeNil())

					router.ServeHTTP(writer, req)

					Expect(writer.Body.String()).To(Equal(oExpected))
					Expect(writer.Code).To(Equal(http.StatusOK))
				},
				Entry("empty slug", "/api/articles//comments", "ArticleCommentList-OK-slug="),
				Entry("normal slug", "/api/articles/s0/comments", "ArticleCommentList-OK-slug=s0"),
			)
		})

		Context("/api/tags - allow anonymous access", func() {
			It("GET _ >> ArticleHandler.TagList", func() {
				mockArticleHandler.EXPECT().TagList(gomock.Any()).Times(1).
					Do(func(c *gin.Context) {
						c.String(http.StatusOK, "TagList-OK")
					})

				router = testApi.SetupRouter()

				req, err = http.NewRequest(http.MethodGet, "/api/tags", nil)
				Expect(err).To(BeNil())

				router.ServeHTTP(writer, req)

				Expect(writer.Body.String()).To(Equal("TagList-OK"))
				Expect(writer.Code).To(Equal(http.StatusOK))
			})
		})
	})

	Context("with AuthMiddleware, require access token", func() {
		BeforeEach(func() {
			mockUserRepo.EXPECT().FindOneUserById(uint(10)).Return(mockLoginUser, nil).AnyTimes()
		})

		Context("/api/user", func() {
			DescribeTable("GET _ >> UsersHandler.UserRetrieve",
				func(iAuth bool, endpointCallCount, oStatus int, oExpected string) {
					mockUsersHandler.EXPECT().UserRetrieve(gomock.Any()).Times(endpointCallCount).
						Do(func(c *gin.Context) {
							c.String(http.StatusOK, "UserRetrieve-OK")
						})

					router = testApi.SetupRouter()

					req, err = http.NewRequest(http.MethodGet, "/api/user", nil)
					Expect(err).To(BeNil())
					if iAuth {
						req.Header.Set(AUTHORIZATION_HEADER, bearerToken)
					}

					router.ServeHTTP(writer, req)

					Expect(writer.Body.String()).To(Equal(oExpected))
					Expect(writer.Code).To(Equal(oStatus))
				},
				Entry("auth", true, 1, http.StatusOK, "UserRetrieve-OK"),
				Entry("no auth", false, 0, http.StatusUnauthorized, ""),
			)

			DescribeTable("PUT _ >> UsersHandler.UserUpdate",
				func(iAuth bool, endpointCallCount, oStatus int, oExpected string) {
					mockUsersHandler.EXPECT().UserUpdate(gomock.Any()).Times(endpointCallCount).
						Do(func(c *gin.Context) {
							bodyBytes, err := io.ReadAll(c.Request.Body)
							Expect(err).To(BeNil())
							c.String(http.StatusOK, fmt.Sprintf("UserUpdate-OK-%s", string(bodyBytes)))
						})

					router = testApi.SetupRouter()

					req, err = http.NewRequest(http.MethodPut, "/api/user", strings.NewReader("requestPayload"))
					Expect(err).To(BeNil())
					if iAuth {
						req.Header.Set(AUTHORIZATION_HEADER, bearerToken)
					}

					router.ServeHTTP(writer, req)

					Expect(writer.Code).To(Equal(oStatus))
					Expect(writer.Body.String()).To(Equal(oExpected))
				},
				Entry("auth", true, 1, http.StatusOK, "UserUpdate-OK-requestPayload"),
				Entry("no auth", false, 0, http.StatusUnauthorized, ""),
			)
		})

		Context("/api/profiles", func() {
			DescribeTable("GET /:username >> UsersHandler.ProfileRetrieve",
				func(iAuth bool, endpointCallCount, oStatus int, oExpected string) {
					mockUsersHandler.EXPECT().ProfileRetrieve(gomock.Any()).Times(endpointCallCount).
						Do(func(c *gin.Context) {
							username := c.Param("username")
							c.String(http.StatusOK, fmt.Sprintf("ProfileRetrieve-OK-username=%s", username))
						})

					router = testApi.SetupRouter()

					req, err = http.NewRequest(http.MethodGet, "/api/profiles/u0", nil)
					Expect(err).To(BeNil())
					if iAuth {
						req.Header.Set(AUTHORIZATION_HEADER, bearerToken)
					}

					router.ServeHTTP(writer, req)

					Expect(writer.Code).To(Equal(oStatus))
					Expect(writer.Body.String()).To(Equal(oExpected))
				},
				Entry("auth", true, 1, http.StatusOK, "ProfileRetrieve-OK-username=u0"),
				Entry("no auth", false, 0, http.StatusUnauthorized, ""),
			)

			DescribeTable("POST /:username/follow >> UsersHandler.ProfileFollow",
				func(iAuth bool, endpointCallCount, oStatus int, oExpected string) {
					mockUsersHandler.EXPECT().ProfileFollow(gomock.Any()).Times(endpointCallCount).
						Do(func(c *gin.Context) {
							username := c.Param("username")
							bodyBytes, err := io.ReadAll(c.Request.Body)
							Expect(err).To(BeNil())
							c.String(http.StatusOK, fmt.Sprintf("ProfileFollow-OK-%s-username=%s", string(bodyBytes), username))
						})

					router = testApi.SetupRouter()

					req, err = http.NewRequest(http.MethodPost, "/api/profiles/u0/follow", strings.NewReader("requestPayload"))
					Expect(err).To(BeNil())
					if iAuth {
						req.Header.Set(AUTHORIZATION_HEADER, bearerToken)
					}

					router.ServeHTTP(writer, req)

					Expect(writer.Code).To(Equal(oStatus))
					Expect(writer.Body.String()).To(Equal(oExpected))
				},
				Entry("auth", true, 1, http.StatusOK, "ProfileFollow-OK-requestPayload-username=u0"),
				Entry("no auth", false, 0, http.StatusUnauthorized, ""),
			)

			DescribeTable("DELETE /:username/follow >> UsersHandler.ProfileUnfollow",
				func(iAuth bool, endpointCallCount, oStatus int, oExpected string) {
					mockUsersHandler.EXPECT().ProfileUnfollow(gomock.Any()).Times(endpointCallCount).
						Do(func(c *gin.Context) {
							username := c.Param("username")
							c.String(http.StatusOK, fmt.Sprintf("ProfileUnfollow-OK-username=%s", username))
						})

					router = testApi.SetupRouter()

					req, err = http.NewRequest(http.MethodDelete, "/api/profiles/u0/follow", nil)
					Expect(err).To(BeNil())
					if iAuth {
						req.Header.Set(AUTHORIZATION_HEADER, bearerToken)
					}

					router.ServeHTTP(writer, req)

					Expect(writer.Code).To(Equal(oStatus))
					Expect(writer.Body.String()).To(Equal(oExpected))
				},
				Entry("auth", true, 1, http.StatusOK, "ProfileUnfollow-OK-username=u0"),
				Entry("no auth", false, 0, http.StatusUnauthorized, ""),
			)
		})

		Context("/api/articles", func() {
			BeforeEach(func() {
				mockUserRepo.EXPECT().FindOneUserById(uint(10)).Return(mockLoginUser, nil).AnyTimes()
			})

			DescribeTable("POST _ >> ArticleHandler.ArticleCreate)",
				func(iAuth bool, endpointCallCount, oStatus int, oExpected string) {
					mockArticleHandler.EXPECT().ArticleCreate(gomock.Any()).Times(endpointCallCount).
						Do(func(c *gin.Context) {
							bodyBytes, err := io.ReadAll(c.Request.Body)
							Expect(err).To(BeNil())
							c.String(http.StatusOK, fmt.Sprintf("ArticleCreate-OK-%s", string(bodyBytes)))
						})

					router = testApi.SetupRouter()

					req, err = http.NewRequest(http.MethodPost, "/api/articles", strings.NewReader("requestPayload"))
					Expect(err).To(BeNil())
					if iAuth {
						req.Header.Set(AUTHORIZATION_HEADER, bearerToken)
					}

					router.ServeHTTP(writer, req)

					Expect(writer.Code).To(Equal(oStatus))
					Expect(writer.Body.String()).To(Equal(oExpected))
				},
				Entry("auth", true, 1, http.StatusOK, "ArticleCreate-OK-requestPayload"),
				Entry("no auth", false, 0, http.StatusUnauthorized, ""),
			)

			DescribeTable("PUT /:slug >> ArticleHandler.ArticleUpdate)",
				func(iAuth bool, endpointCallCount, oStatus int, oExpected string) {
					mockArticleHandler.EXPECT().ArticleUpdate(gomock.Any()).Times(endpointCallCount).
						Do(func(c *gin.Context) {
							slug := c.Param("slug")
							bodyBytes, err := io.ReadAll(c.Request.Body)
							Expect(err).To(BeNil())
							c.String(http.StatusOK, fmt.Sprintf("ArticleUpdate-OK-%s-slug=%s", string(bodyBytes), slug))
						})

					router = testApi.SetupRouter()

					req, err = http.NewRequest(http.MethodPut, "/api/articles/s0", strings.NewReader("requestPayload"))
					Expect(err).To(BeNil())
					if iAuth {
						req.Header.Set(AUTHORIZATION_HEADER, bearerToken)
					}

					router.ServeHTTP(writer, req)

					Expect(writer.Code).To(Equal(oStatus))
					Expect(writer.Body.String()).To(Equal(oExpected))
				},
				Entry("auth", true, 1, http.StatusOK, "ArticleUpdate-OK-requestPayload-slug=s0"),
				Entry("no auth", false, 0, http.StatusUnauthorized, ""),
			)

			DescribeTable("DELETE /:slug >> ArticleHandler.ArticleDelete)",
				func(iAuth bool, endpointCallCount, oStatus int, oExpected string) {
					mockArticleHandler.EXPECT().ArticleDelete(gomock.Any()).Times(endpointCallCount).
						Do(func(c *gin.Context) {
							slug := c.Param("slug")
							c.String(http.StatusOK, fmt.Sprintf("ArticleDelete-OK-slug=%s", slug))
						})

					router = testApi.SetupRouter()

					req, err = http.NewRequest(http.MethodDelete, "/api/articles/s0", nil)
					Expect(err).To(BeNil())
					if iAuth {
						req.Header.Set(AUTHORIZATION_HEADER, bearerToken)
					}

					router.ServeHTTP(writer, req)

					Expect(writer.Code).To(Equal(oStatus))
					Expect(writer.Body.String()).To(Equal(oExpected))
				},
				Entry("auth", true, 1, http.StatusOK, "ArticleDelete-OK-slug=s0"),
				Entry("no auth", false, 0, http.StatusUnauthorized, ""),
			)

			DescribeTable("POST /:slug/favorite >> ArticleHandler.ArticleFavorite)",
				func(iAuth bool, endpointCallCount, oStatus int, oExpected string) {
					mockArticleHandler.EXPECT().ArticleFavorite(gomock.Any()).Times(endpointCallCount).
						Do(func(c *gin.Context) {
							slug := c.Param("slug")
							bodyBytes, err := io.ReadAll(c.Request.Body)
							Expect(err).To(BeNil())
							c.String(http.StatusOK, fmt.Sprintf("ArticleFavorite-OK-%s-slug=%s", string(bodyBytes), slug))
						})

					router = testApi.SetupRouter()

					req, err = http.NewRequest(http.MethodPost, "/api/articles/s0/favorite", strings.NewReader("requestPayload"))
					Expect(err).To(BeNil())
					if iAuth {
						req.Header.Set(AUTHORIZATION_HEADER, bearerToken)
					}

					router.ServeHTTP(writer, req)

					Expect(writer.Code).To(Equal(oStatus))
					Expect(writer.Body.String()).To(Equal(oExpected))
				},
				Entry("auth", true, 1, http.StatusOK, "ArticleFavorite-OK-requestPayload-slug=s0"),
				Entry("no auth", false, 0, http.StatusUnauthorized, ""),
			)

			DescribeTable("DELETE /:slug/favorite >> ArticleHandler.ArticleUnfavorite)",
				func(iAuth bool, endpointCallCount, oStatus int, oExpected string) {
					mockArticleHandler.EXPECT().ArticleUnfavorite(gomock.Any()).Times(endpointCallCount).
						Do(func(c *gin.Context) {
							slug := c.Param("slug")
							c.String(http.StatusOK, fmt.Sprintf("ArticleUnfavorite-OK-slug=%s", slug))
						})

					router = testApi.SetupRouter()

					req, err = http.NewRequest(http.MethodDelete, "/api/articles/s0/favorite", nil)
					Expect(err).To(BeNil())
					if iAuth {
						req.Header.Set(AUTHORIZATION_HEADER, bearerToken)
					}

					router.ServeHTTP(writer, req)

					Expect(writer.Code).To(Equal(oStatus))
					Expect(writer.Body.String()).To(Equal(oExpected))
				},
				Entry("auth", true, 1, http.StatusOK, "ArticleUnfavorite-OK-slug=s0"),
				Entry("no auth", false, 0, http.StatusUnauthorized, ""),
			)

			DescribeTable("POST /:slug/comments >> ArticleHandler.ArticleCommentCreate)",
				func(iAuth bool, endpointCallCount, oStatus int, oExpected string) {
					mockArticleHandler.EXPECT().ArticleCommentCreate(gomock.Any()).Times(endpointCallCount).
						Do(func(c *gin.Context) {
							slug := c.Param("slug")
							bodyBytes, err := io.ReadAll(c.Request.Body)
							Expect(err).To(BeNil())
							c.String(http.StatusOK, fmt.Sprintf("ArticleCommentCreate-OK-%s-slug=%s", string(bodyBytes), slug))
						})

					router = testApi.SetupRouter()

					req, err = http.NewRequest(http.MethodPost, "/api/articles/s0/comments", strings.NewReader("requestPayload"))
					Expect(err).To(BeNil())
					if iAuth {
						req.Header.Set(AUTHORIZATION_HEADER, bearerToken)
					}

					router.ServeHTTP(writer, req)

					Expect(writer.Code).To(Equal(oStatus))
					Expect(writer.Body.String()).To(Equal(oExpected))
				},
				Entry("auth", true, 1, http.StatusOK, "ArticleCommentCreate-OK-requestPayload-slug=s0"),
				Entry("no auth", false, 0, http.StatusUnauthorized, ""),
			)

			DescribeTable("DELETE /:slug/comments/:id >> ArticleHandler.ArticleCommentDelete)",
				func(iAuth bool, endpointCallCount, oStatus int, oExpected string) {
					mockArticleHandler.EXPECT().ArticleCommentDelete(gomock.Any()).Times(endpointCallCount).
						Do(func(c *gin.Context) {
							slug := c.Param("slug")
							id := c.Param("id")
							c.String(http.StatusOK, fmt.Sprintf("ArticleCommentDelete-OK-slug=%s-id=%s", slug, id))
						})

					router = testApi.SetupRouter()

					req, err = http.NewRequest(http.MethodDelete, "/api/articles/s0/comments/i0", nil)
					Expect(err).To(BeNil())
					if iAuth {
						req.Header.Set(AUTHORIZATION_HEADER, bearerToken)
					}

					router.ServeHTTP(writer, req)

					Expect(writer.Code).To(Equal(oStatus))
					Expect(writer.Body.String()).To(Equal(oExpected))
				},
				Entry("auth", true, 1, http.StatusOK, "ArticleCommentDelete-OK-slug=s0-id=i0"),
				Entry("no auth", false, 0, http.StatusUnauthorized, ""),
			)
		})
	})

})
