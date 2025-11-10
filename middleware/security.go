package middleware

import (
	"com/realworld/ginrxgogorm/repository"
	"errors"
	"time"

	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/bcrypt"
)

const (
	TOKEN_SECRET                 = "+*+ a Secret String >4< the token generation #!#"
	CURRENT_USER_GIN_VAR_NAME    = "my_user_model"
	CURRENT_USER_ID_GIN_VAR_NAME = "my_user_id"
	USER_ID_JWT_CLAIM_NAME       = "id"
)

func GenToken(id uint) string {
	jwt_token := jwt.New(jwt.GetSigningMethod("HS256"))
	// remember the login user
	jwt_token.Claims = jwt.MapClaims{
		USER_ID_JWT_CLAIM_NAME: id,
		"exp":                  time.Now().Add(time.Hour * 24).Unix(),
	}
	token, _ := jwt_token.SignedString([]byte(TOKEN_SECRET))
	return token
}

func SetPassword(u *repository.UserModel, password string) (repository.UserModel, error) {
	if len(password) == 0 {
		return *u, errors.New("password should not be empty!")
	}
	bytePassword := []byte(password)
	// Make sure the second param `bcrypt generator cost` between [4, 32)
	passwordHash, _ := bcrypt.GenerateFromPassword(bytePassword, bcrypt.DefaultCost)
	u.PasswordHash = string(passwordHash)
	return *u, nil
}

func CheckPassword(u *repository.UserModel, password string) error {
	bytePassword := []byte(password)
	byteHashedPassword := []byte(u.PasswordHash)
	return bcrypt.CompareHashAndPassword(byteHashedPassword, bytePassword)
}
