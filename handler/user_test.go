package handler_test

import (
	"com/realworld/ginrxgogorm/handler"
	"com/realworld/ginrxgogorm/middleware"
	"com/realworld/ginrxgogorm/repository"
	mock_repository "com/realworld/ginrxgogorm/repository/mock"
	"com/realworld/ginrxgogorm/test_data"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func setLoginUser(gc *gin.Context, loginUser *repository.UserModel) {
	if loginUser != nil {
		gc.Set(middleware.CURRENT_USER_GIN_VAR_NAME, *loginUser)
		return
	}
	gc.Set(middleware.CURRENT_USER_GIN_VAR_NAME, nil)
}

var _ = Describe("Testing End Points", func() {
	const (
		correlationId = "anyCorrelationId"
	)

	var (
		writer       *httptest.ResponseRecorder
		mockUserRepo *mock_repository.MockUsersRepository
		gCtx         *gin.Context
		testHandler  handler.UsersHandler
		reqTemplate  = http.Request{
			Header: http.Header{
				"Content-Type":     []string{"application/json"},
				"X-Correlation-ID": []string{correlationId},
			},
			Method: http.MethodPost,
		}
	)

	var ctrl *gomock.Controller
	BeforeEach(func() {
		// Skip("generic")
		ctrl = gomock.NewController(GinkgoT())
		mockUserRepo = mock_repository.NewMockUsersRepository(ctrl)
		testHandler = handler.NewUsersHandler(mockUserRepo)
		writer = httptest.NewRecorder()
		gCtx, _ = gin.CreateTestContext(writer)
	})
	AfterEach(func() {
		ctrl.Finish()
	})

	Context("/users", func() {
		Context("POST /~ >> UserHandler.UsersRegistration()", func() {
			DescribeTable("testing...",
				func(requestPayload string, setupMock func(), oHttpStatus int, oResPayloadStr string) {
					req := reqTemplate
					req.Body = io.NopCloser(strings.NewReader(requestPayload))
					gCtx.Request = &req
					setupMock()

					testHandler.UsersRegistration(gCtx)

					Expect(writer.Code).To(Equal(oHttpStatus))
					Expect(writer.Body.String()).To(ContainSubstring(oResPayloadStr))
				},
				Entry("should return StatusCreated, if user registration succeed",
					`{ "user":
						{ "username": "username0",
						  "email": "test@t.t",
						  "password": "password0"
						}
					}`,
					func() {
						mockUserRepo.EXPECT().SaveOne(gomock.Cond(func(arg any) bool {
							val := arg.(repository.UserModel)
							return val.Username == "username0" && val.Email == "test@t.t"
						})).Return(repository.UserModel{}, nil).Times(1)
					},
					http.StatusCreated,
					`{"user":{"username":"username0","email":"test@t.t","token":"`),
				Entry("should return StatusBadRequest, if username is too short",
					`{ "user":
						{ "username": "us",
						  "email": "test@t.t",
						  "password": "password0"
						}
					}`,
					func() {},
					http.StatusBadRequest,
					`{"errors":{"Username":"{min: 4}"}}`),
				Entry("should return StatusBadRequest, if username contains other character",
					`{ "user":
						{ "username": "us---------",
						  "email": "test@t.t",
						  "password": "password0"
						}
					}`,
					func() {},
					http.StatusBadRequest,
					`{"errors":{"Username":"{key: alphanum}"}}`),
				Entry("should return StatusBadRequest, if required username is missing",
					`{ "user":
						{ "email": "test@t.t",
						  "password": "password0"
						}
					}`,
					func() {},
					http.StatusBadRequest,
					`{"errors":{"Username":"{key: required}"}}`),
				Entry("should return StatusBadRequest, if required email is missing",
					`{ "user":
						{ "username": "username0",
						  "password": "password0"
						}
					}`,
					func() {},
					http.StatusBadRequest,
					`{"errors":{"Email":"{key: required}"}}`),
				Entry("should return StatusBadRequest, if required password is missing",
					`{ "user":
						{ "username": "username0",
						  "email": "test@t.t"
						}
					}`,
					func() {},
					http.StatusBadRequest,
					`{"errors":{"Password":"{key: required}"}}`),
				Entry("should return StatusBadRequest, if the request has multiple issues",
					`{ "user":
						{ "username": "us",
						  "password": "password0"
						}
					}`,
					func() {},
					http.StatusBadRequest,
					`{"errors":{"Email":"{key: required}","Username":"{min: 4}"}}`),
				Entry("should return StatusUnprocessableEntity, when save user to db failed",
					`{ "user":
						{ "username": "username0",
						  "email": "test@t.t",
						  "password": "password0"
						}
					}`,
					func() {
						mockUserRepo.EXPECT().SaveOne(gomock.Any()).Return(repository.UserModel{}, errors.New("some error")).Times(1)
					},
					http.StatusUnprocessableEntity,
					`{"errors":{"database":"some error"}}`),
			)
		})

		Context("POST /login >> UserHandler.UsersLogin()", func() {
			var passwordHash []byte
			DescribeTable("testing...",
				func(requestPayload string, setupMock func(), oHttpStatus int, oResPayloadStr string) {
					req := reqTemplate
					req.Body = io.NopCloser(strings.NewReader(requestPayload))
					gCtx.Request = &req
					setupMock()

					testHandler.UsersLogin(gCtx)

					Expect(writer.Code).To(Equal(oHttpStatus))
					Expect(writer.Body.String()).To(ContainSubstring(oResPayloadStr))
				},
				Entry("should return StatusOK, if user login succeed",
					`{ "user":
						{ "email": "test@t.t",
						  "password": "password0"
						}
					}`,
					func() {
						passwordHash, _ = bcrypt.GenerateFromPassword([]byte("password0"), bcrypt.DefaultCost)
						mockUserRepo.EXPECT().FindOneUser(gomock.Any()).Return(repository.UserModel{
							PasswordHash: string(passwordHash),
						}, nil).Times(1)
					},
					http.StatusOK,
					`{"user":{"token":"`),
				Entry("should return StatusForbidden, when missing required email",
					`{ "user":
						{ "password": "password0" }
					}`,
					func() {},
					http.StatusBadRequest,
					`{"errors":{"Email":"{key: required}"}}`),
				Entry("should return StatusForbidden, if cannot find user",
					`{ "user":
						{ "email": "test@t.t",
						  "password": "password0"
						}
					}`,
					func() {
						mockUserRepo.EXPECT().FindOneUser(gomock.Any()).Return(repository.UserModel{}, nil).Times(1)
					},
					http.StatusForbidden,
					`{"errors":{"login":"Not Registered email or invalid password"}}`),
				Entry("should return StatusForbidden, if password mismatch",
					`{ "user":
						{ "email": "test@t.t",
						  "password": "password0"
						}
					}`,
					func() {
						mockUserRepo.EXPECT().FindOneUser(gomock.Any()).Return(repository.UserModel{
							PasswordHash: "some random value",
						}, nil).Times(1)
					},
					http.StatusForbidden,
					`{"errors":{"login":"Not Registered email or invalid password"}}`),
				Entry("should return StatusForbidden, if db call failed",
					`{ "user":
						{ "email": "test@t.t",
						  "password": "password0"
						}
					}`,
					func() {
						mockUserRepo.EXPECT().FindOneUser(gomock.Any()).Return(repository.UserModel{}, errors.New("some db error")).Times(1)
					},
					http.StatusForbidden,
					`{"errors":{"login":"Not Registered email or invalid password"}}`),
			)
		})
	})

	Context("/user", func() {
		Context("GET /~ >> UserHandler.UserRetrieve()", func() {
			It("should return the current login user in the gin context", func() {
				lgCtx, _ := gin.CreateTestContext(writer)
				setLoginUser(lgCtx, &repository.UserModel{
					Username:     "Username",
					Email:        "Email",
					Bio:          "Bio",
					Image:        &test_data.Test_Img,
					PasswordHash: "PasswordHash",
					Followings:   test_data.GenerateTestUsers(0, 3, false),
				})

				testHandler.UserRetrieve(lgCtx)

				Expect(writer.Code).To(Equal(http.StatusOK))
				Expect(writer.Body.String()).To(ContainSubstring(
					`{"user":{"username":"Username","email":"Email","bio":"Bio","image":"some random image","token":"`))
			})

			It("should return the empty user, if gin Context cannot find a valid current user", func() {
				lgCtx, _ := gin.CreateTestContext(writer)
				setLoginUser(lgCtx, nil)

				testHandler.UserRetrieve(lgCtx)

				Expect(writer.Code).To(Equal(http.StatusOK))
				Expect(writer.Body.String()).To(ContainSubstring(
					`{"user":{"token":"`))
			})
		})

		Context("PUT /~ >> UserHandler.UserUpdate()", func() {
			testOrgLoginUser := repository.UserModel{
				Model:        gorm.Model{ID: 5},
				Username:     "Username",
				Email:        "Email",
				Bio:          "Bio",
				Image:        &test_data.Test_Img,
				PasswordHash: "PasswordHash",
				Followings:   test_data.GenerateTestUsers(0, 3, false),
			}
			var testLoginUser repository.UserModel
			BeforeEach(func() {
				testLoginUser = testOrgLoginUser
				setLoginUser(gCtx, &testLoginUser)
			})
			DescribeTable("testing...",
				func(requestPayload string, setupMock func(), oHttpStatus int, oResPayloadStr string) {
					req := reqTemplate
					req.Body = io.NopCloser(strings.NewReader(requestPayload))
					gCtx.Request = &req
					setupMock()

					testHandler.UserUpdate(gCtx)

					Expect(writer.Body.String()).To(ContainSubstring(oResPayloadStr))
					Expect(writer.Code).To(Equal(oHttpStatus))
				},
				Entry("should return StatusOK, if user updated succeed",
					`{ "user":
						{ "username": "username1",
						  "password": "password0"
						}
					}`,
					func() {
						mockUserRepo.EXPECT().Update(gomock.Any(), gomock.Cond(func(arg any) bool {
							val := arg.(repository.UserModel)
							return val.Username == "username1" && val.Email == "Email"
						})).Return(repository.UserModel{}, nil).Times(1)
						updatedLoginUser := testOrgLoginUser
						updatedLoginUser.Username = "username1"
						mockUserRepo.EXPECT().FindOneUserById(uint(5)).Return(updatedLoginUser, nil).Times(1)
					},
					http.StatusOK,
					`{"user":{"username":"username1","email":"Email","bio":"Bio","image":"some random image","token":"`),
				Entry("should return StatusBadRequest, if username is too short",
					`{ "user":
						{ "username": "us",
						  "email": "test@t.t"
						}
					}`,
					func() {},
					http.StatusBadRequest,
					`{"errors":{"Username":"{min: 4}"}}`),
				Entry("should return StatusUnprocessableEntity, when save user to db failed",
					`{ "user":
						{ "username": "username1",
						  "password": "password0"
						}
					}`,
					func() {
						mockUserRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(repository.UserModel{}, errors.New("some error")).Times(1)
					},
					http.StatusUnprocessableEntity,
					`{"errors":{"database":"some error"}}`),
			)
		})
	})

	Context("/profiles", func() {
		Context("GET /:username >> UserHandler.ProfileRetrieve()", func() {
			DescribeTable("testing...",
				func(setupMock func(), oHttpStatus int, oResPayloadStr string) {
					req := reqTemplate
					gCtx.Request = &req
					gCtx.Params = []gin.Param{
						{Key: "username", Value: "test_user"},
					}
					setLoginUser(gCtx, &repository.UserModel{})

					setupMock()

					testHandler.ProfileRetrieve(gCtx)

					Expect(writer.Code).To(Equal(oHttpStatus))
					Expect(writer.Body.String()).To(ContainSubstring(oResPayloadStr))
				},
				Entry("should return StatusOK, if user retrieve profile succeed",
					func() {
						mockUserRepo.EXPECT().FindOneUser(gomock.Any()).Return(test_data.GenerateTestUsers(0, 1, false)[0], nil).Times(1)
						mockUserRepo.EXPECT().IsFollowings(gomock.Any(), gomock.Any()).Return(true, nil).Times(1)
					},
					http.StatusOK,
					`{"profile":{"username":"user1","bio":"bio1","image":"http://image/1.jpg","following":true}}`),
				Entry("should return StatusOK, when error on checking followings",
					func() {
						mockUserRepo.EXPECT().FindOneUser(gomock.Any()).Return(test_data.GenerateTestUsers(0, 1, false)[0], nil).Times(1)
						mockUserRepo.EXPECT().IsFollowings(gomock.Any(), gomock.Any()).Return(false, errors.New("some error")).Times(1)
					},
					http.StatusOK,
					`{"profile":{"username":"user1","bio":"bio1","image":"http://image/1.jpg","following":false}}`),
				Entry("should return StatusForbidden, if cannot find user",
					func() {
						mockUserRepo.EXPECT().FindOneUser(gomock.Any()).Return(repository.UserModel{}, errors.New("some error")).Times(1)
					},
					http.StatusNotFound,
					`{"errors":{"profile":"invalid username"}}`),
			)
		})

		Context("POST /:username/follow >> UserHandler.ProfileFollow()", func() {
			DescribeTable("testing...",
				func(setupMock func(), oHttpStatus int, oResPayloadStr string) {
					req := reqTemplate
					gCtx.Request = &req
					gCtx.Params = []gin.Param{
						{Key: "username", Value: "test_user"},
					}
					setLoginUser(gCtx, &repository.UserModel{})

					setupMock()

					testHandler.ProfileFollow(gCtx)

					Expect(writer.Code).To(Equal(oHttpStatus))
					Expect(writer.Body.String()).To(ContainSubstring(oResPayloadStr))
				},
				Entry("should return StatusOK, if user retrieve profile succeed",
					func() {
						mockUserRepo.EXPECT().FindOneUser(gomock.Any()).Return(test_data.GenerateTestUsers(0, 1, false)[0], nil).Times(1)
						mockUserRepo.EXPECT().Following(gomock.Any(), gomock.Any()).Return(nil).Times(1)
					},
					http.StatusOK,
					`{"profile":{"username":"user1","bio":"bio1","image":"http://image/1.jpg","following":true}}`),
				Entry("should return StatusOK, when error on following",
					func() {
						mockUserRepo.EXPECT().FindOneUser(gomock.Any()).Return(test_data.GenerateTestUsers(0, 1, false)[0], nil).Times(1)
						mockUserRepo.EXPECT().Following(gomock.Any(), gomock.Any()).Return(errors.New("some error")).Times(1)
					},
					http.StatusUnprocessableEntity,
					`{"errors":{"database":"some error"}}`),
				Entry("should return StatusForbidden, if cannot find user",
					func() {
						mockUserRepo.EXPECT().FindOneUser(gomock.Any()).Return(repository.UserModel{}, errors.New("some error")).Times(1)
					},
					http.StatusNotFound,
					`{"errors":{"profile":"invalid username"}}`),
			)
		})

		Context("DELETE /:username/follow_ >> UserHandler.ProfileUnfollow()", func() {
			DescribeTable("testing...",
				func(setupMock func(), oHttpStatus int, oResPayloadStr string) {
					req := reqTemplate
					gCtx.Request = &req
					gCtx.Params = []gin.Param{
						{Key: "username", Value: "test_user"},
					}
					setLoginUser(gCtx, &repository.UserModel{})

					setupMock()

					testHandler.ProfileUnfollow(gCtx)

					Expect(writer.Code).To(Equal(oHttpStatus))
					Expect(writer.Body.String()).To(ContainSubstring(oResPayloadStr))
				},
				Entry("should return StatusOK, if user retrieve profile succeed",
					func() {
						mockUserRepo.EXPECT().FindOneUser(gomock.Any()).Return(test_data.GenerateTestUsers(0, 1, false)[0], nil).Times(1)
						mockUserRepo.EXPECT().Unfollowing(gomock.Any(), gomock.Any()).Return(nil).Times(1)
					},
					http.StatusOK,
					`{"profile":{"username":"user1","bio":"bio1","image":"http://image/1.jpg","following":false}}`),
				Entry("should return StatusOK, when error on following",
					func() {
						mockUserRepo.EXPECT().FindOneUser(gomock.Any()).Return(test_data.GenerateTestUsers(0, 1, false)[0], nil).Times(1)
						mockUserRepo.EXPECT().Unfollowing(gomock.Any(), gomock.Any()).Return(errors.New("some error")).Times(1)
					},
					http.StatusUnprocessableEntity,
					`{"errors":{"database":"some error"}}`),
				Entry("should return StatusForbidden, if cannot find user",
					func() {
						mockUserRepo.EXPECT().FindOneUser(gomock.Any()).Return(repository.UserModel{}, errors.New("some error")).Times(1)
					},
					http.StatusNotFound,
					`{"errors":{"profile":"invalid username"}}`),
			)
		})
	})
})
