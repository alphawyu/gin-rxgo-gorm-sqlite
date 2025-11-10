package test_data

import (
	"com/realworld/ginrxgogorm/middleware"
	"com/realworld/ginrxgogorm/repository"
	"fmt"

	log "github.com/sirupsen/logrus"

	"gorm.io/gorm"
)

const TEST_DB_FILE = "./gorm_test.db"
// const TEST_DB_FILE = ""

var Test_Img = "some random image"

// CAUTION: this func really add the UserModel to the DB!
func UserModelMocker(test_db *gorm.DB, n int64) []repository.UserModel {
	var offset int64
	test_db.Model(&repository.UserModel{}).Count(&offset)
	ret := GenerateTestUsers(offset, n, false)
	result := test_db.Create(&ret)
	if result.Error != nil {
		log.Printf("error %v", result.Error)
	}

	return ret
}

func GenerateTestUsers(offset, n int64, withId bool) []repository.UserModel {
	var ret []repository.UserModel
	for i := offset + 1; i <= offset+n; i++ {
		image := fmt.Sprintf("http://image/%v.jpg", i)
		userModel := repository.UserModel{
			Username: fmt.Sprintf("user%v", i),
			Email:    fmt.Sprintf("user%v@linkedin.com", i),
			Bio:      fmt.Sprintf("bio%v", i),
			Image:    &image,
		}
		if withId {
			userModel.ID = uint(i)
		}
 		middleware.SetPassword(&userModel, "password123")
		ret = append(ret, userModel)
	}
	return ret
}
