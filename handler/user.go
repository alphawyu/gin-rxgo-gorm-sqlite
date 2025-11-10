package handler

import (
	"com/realworld/ginrxgogorm/middleware"
	"com/realworld/ginrxgogorm/model"
	"com/realworld/ginrxgogorm/repository"
	"com/realworld/ginrxgogorm/util"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:generate mockgen -destination mock/users_mock.go -source user_handler.go UsersHandler
type UsersHandler interface {
	UsersRegistration(c *gin.Context)
	UsersLogin(c *gin.Context)
	UserRetrieve(c *gin.Context)
	UserUpdate(c *gin.Context)

	ProfileRetrieve(c *gin.Context)
	ProfileFollow(c *gin.Context)
	ProfileUnfollow(c *gin.Context)
}

type UsersHandlerImpl struct {
	UsersRepo repository.UsersRepository
}

func NewUsersHandler(usersRepo repository.UsersRepository) *UsersHandlerImpl {
	return &UsersHandlerImpl{
		UsersRepo: usersRepo,
	}
}

// to ensure the __impl implements the interface at compile-time
var _ UsersHandler = &UsersHandlerImpl{}

func (a *UsersHandlerImpl) ProfileRetrieve(c *gin.Context) {
	username := c.Param("username")
	userModel, err := a.UsersRepo.FindOneUser(repository.UserModel{Username: username})
	if err != nil {
		c.JSON(http.StatusNotFound, util.NewError("profile", errors.New("invalid username")))
		return
	}
	followingFlag, _ := a.UsersRepo.IsFollowings(getLoginUser(c), userModel)

	response := makeUserProfileResponse(userModel, followingFlag)
	c.JSON(http.StatusOK, gin.H{"profile": response})
}

func (a *UsersHandlerImpl) ProfileFollow(c *gin.Context) {
	username := c.Param("username")
	userModel, err := a.UsersRepo.FindOneUser(repository.UserModel{Username: username})
	if err != nil {
		c.JSON(http.StatusNotFound, util.NewError("profile", errors.New("invalid username")))
		return
	}
	err = a.UsersRepo.Following(getLoginUser(c), userModel)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, util.NewError("database", err))
		return
	}
	response := makeUserProfileResponse(userModel, true)
	c.JSON(http.StatusOK, gin.H{"profile": response})
}

func (a *UsersHandlerImpl) ProfileUnfollow(c *gin.Context) {
	username := c.Param("username")
	userModel, err := a.UsersRepo.FindOneUser(repository.UserModel{Username: username})
	if err != nil {
		c.JSON(http.StatusNotFound, util.NewError("profile", errors.New("invalid username")))
		return
	}

	err = a.UsersRepo.Unfollowing(getLoginUser(c), userModel)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, util.NewError("database", err))
		return
	}
	response := makeUserProfileResponse(userModel, false)
	c.JSON(http.StatusOK, gin.H{"profile": response})
}

func (a *UsersHandlerImpl) UsersRegistration(c *gin.Context) {
	var uw model.UserWrapper
	if err := c.BindJSON(&uw); err != nil {
		c.JSON(http.StatusUnprocessableEntity, util.NewValidatorError(err))
		return
	}
	user := uw.User
	userModel := repository.UserModel{
		Username: user.Username,
		Email:    user.Email,
		Bio:      user.Bio,
		Image:    user.Image,
	}
	middleware.SetPassword(&userModel, user.Password)

	if _, err := a.UsersRepo.SaveOne(userModel); err != nil {
		c.JSON(http.StatusUnprocessableEntity, util.NewError("database", err))
		return
	}
	c.Set(middleware.CURRENT_USER_GIN_VAR_NAME, userModel)
	response := makeUserResponse(getLoginUser(c))
	c.JSON(http.StatusCreated, gin.H{"user": response})
}

func (a *UsersHandlerImpl) UserUpdate(c *gin.Context) {
	var uuw model.UpdateUserWrapper
	if err := c.BindJSON(&uuw); err != nil {
		c.JSON(http.StatusUnprocessableEntity, util.NewValidatorError(err))
		return
	}
	user := uuw.User
	loginUser := getLoginUser(c)
	userModel := repository.UserModel{
		Username:     util.UserUpdateIfNotEmpty(loginUser.Username, user.Username),
		Email:        util.UserUpdateIfNotEmpty(loginUser.Email, user.Email),
		Bio:          util.UserUpdateIfNotEmpty(loginUser.Bio, user.Bio),
		Image:        util.UserUpdateIfNotNil(loginUser.Image, user.Image),
		PasswordHash: loginUser.PasswordHash,
	}
	if user.Password != "" {
		middleware.SetPassword(&userModel, user.Password)
	}
	userModel.ID = loginUser.ID
	if _, err := a.UsersRepo.Update(loginUser, userModel); err != nil {
		c.JSON(http.StatusUnprocessableEntity, util.NewError("database", err))
		return
	}
	middleware.UpdateContextUserModel(c, a.UsersRepo, loginUser.ID)
	response := makeUserResponse(getLoginUser(c))
	c.JSON(http.StatusOK, gin.H{"user": response})
}

func (a *UsersHandlerImpl) UsersLogin(c *gin.Context) {
	var luw model.LoginUserWrapper
	if err := c.BindJSON(&luw); err != nil {
		c.JSON(http.StatusUnprocessableEntity, util.NewValidatorError(err))
		return
	}
	user := luw.LoginUser
	userModel := repository.UserModel{
		Email: user.Email,
	}

	userModel, err := a.UsersRepo.FindOneUser(userModel)
	if err != nil {
		c.JSON(http.StatusForbidden, util.NewError("login", util.LOGIN_ERROR))
		return
	}

	if middleware.CheckPassword(&userModel, user.Password) != nil {
		c.JSON(http.StatusForbidden, util.NewError("login", util.LOGIN_ERROR))
		return
	}
	middleware.UpdateContextUserModel(c, a.UsersRepo, userModel.ID)
	response := makeUserResponse(getLoginUser(c))
	c.JSON(http.StatusOK, gin.H{"user": response})
}

func (a *UsersHandlerImpl) UserRetrieve(c *gin.Context) {
	response := makeUserResponse(getLoginUser(c))
	c.JSON(http.StatusOK, gin.H{"user": response})
}

// --------------------------------------------------------------------------------
func getLoginUser(gc *gin.Context) repository.UserModel {
	if loginUserAny, exists := gc.Get(middleware.CURRENT_USER_GIN_VAR_NAME); exists {
		if loginUser, ok := loginUserAny.(repository.UserModel); ok {
			return loginUser
		}
	}

	return repository.UserModel{}
}

func makeUserResponse(loginUser repository.UserModel) model.UserResponse {
	user := model.UserResponse{
		Username: loginUser.Username,
		Email:    loginUser.Email,
		Bio:      loginUser.Bio,
		Image:    loginUser.Image,
		Token:    middleware.GenToken(loginUser.ID),
	}
	return user
}
