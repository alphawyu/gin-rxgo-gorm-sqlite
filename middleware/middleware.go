package middleware

import (
	"com/realworld/ginrxgogorm/repository"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/dgrijalva/jwt-go/request"
	"github.com/gin-gonic/gin"
)

// Strips 'TOKEN ' prefix from token string
func stripBearerPrefixFromTokenString(tok string) (string, error) {
	// Should be a bearer token
	if len(tok) > 5 && strings.ToUpper(tok[0:6]) == "TOKEN " {
		return tok[6:], nil
	}
	return tok, nil
}

// Extract  token from Authorization header
// Uses PostExtractionFilter to strip "TOKEN " prefix from header
var AuthorizationHeaderExtractor = &request.PostExtractionFilter{
	Extractor: request.HeaderExtractor{"Authorization"},
	Filter: stripBearerPrefixFromTokenString,
}

// Extractor for OAuth2 access tokens.  Looks in 'Authorization'
// header then 'access_token' argument for a token.
var MyAuth2Extractor = &request.MultiExtractor{
	AuthorizationHeaderExtractor,
	request.ArgumentExtractor{"access_token"},
}

// A helper to write user_id and user_model to the context
func UpdateContextUserModel(c *gin.Context, userRepo repository.UsersRepository, myUserId uint) {
	var myUserModel repository.UserModel
	if myUserId != 0 {
		myUserModel, _ = userRepo.FindOneUserById(myUserId)
	}
	c.Set(CURRENT_USER_ID_GIN_VAR_NAME, myUserId)
	c.Set(CURRENT_USER_GIN_VAR_NAME, myUserModel)
}

// You can custom middlewares yourself as the doc: https://github.com/gin-gonic/gin#custom-middleware
//
//	r.Use(AuthMiddleware(true))
func AuthMiddleware(userRepo repository.UsersRepository, auto401 bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		UpdateContextUserModel(c, userRepo, 0)
		token, err := request.ParseFromRequest(c.Request, MyAuth2Extractor,
			// keyFunc provide the key for jwt verify the signature, can be dynamically created on the token detail
			func(token *jwt.Token) (any, error) { return []byte(TOKEN_SECRET), nil })
		if err != nil {
			log.Errorf("AuthMiddleware:ParseFromRequest error %v", err)
			if auto401 {
				c.AbortWithError(http.StatusUnauthorized, err)
			}
			return
		}
		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			myUserId := uint(claims[USER_ID_JWT_CLAIM_NAME].(float64))
			//fmt.Println(my_user_id,claims[USER_ID_JWT_CLAIM_NAME])
			UpdateContextUserModel(c, userRepo, myUserId)
		}
	}
}
