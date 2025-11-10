package test_data

import (
	repository "com/realworld/ginrxgogorm/repository"
	"fmt"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// const TEST_DB_FILE = "./../gorm_test.db"

func ArticleModelMocker(test_db *gorm.DB, n int64) []repository.ArticleModel {
	users := UserModelMocker(test_db, n)

	var offset int64
	test_db.Model(&repository.ArticleModel{}).Count(&offset)
	var ret []repository.ArticleModel
	for i := offset + 1; i <= offset+n; i++ {
		articleModel := repository.ArticleModel{
			Slug:        fmt.Sprintf("Slug-%d", i),
			Title:       fmt.Sprintf("Title-%d", i),
			Description: fmt.Sprintf("Desc-%d", i),
			Body:        fmt.Sprintf("Body-%d", i),
			AuthorID: users[i].ID,
		}
		result := test_db.Create(&articleModel)
		if result.Error != nil {
			log.Printf("error %v", result.Error)
		}
		ret = append(ret, articleModel)
	}
	return ret
}


func GenerateTestArticles(offset, n int64, withId bool, u []repository.UserModel) []repository.ArticleModel {
	users := u
	lu := int64(len(users))
	if lu < 1 {
		u2 := GenerateTestUsers(offset, 1, withId)
		users = append(users, u2...)	
		lu = 1	
	}

	var ret []repository.ArticleModel
	for i := 1; i <= int(n); i++ {
		ai := int64(i) + offset
		ii := i % int(lu)
		articleModel := repository.ArticleModel{
			Slug:        fmt.Sprintf("Slug-%d", ai),
			Title:       fmt.Sprintf("Title-%d", ai),
			Description: fmt.Sprintf("Desc-%d", ai),
			Body:        fmt.Sprintf("Body-%d", ai),
			Author: repository.ArticleUserModel{
				UserModel:   users[ii],
				UserModelID: users[ii].ID,
			},
			AuthorID: users[ii].ID,
			Tags:        []repository.TagModel{
				{ Tag: fmt.Sprintf("Tag1-%d", ai) },
				{ Tag: fmt.Sprintf("Tag2-%d", ai) },
			},
			Comments:    []repository.CommentModel{
				{ Body: fmt.Sprintf("Cb1-%d", ai), AuthorID: users[lu-1].ID },
				{ Body: fmt.Sprintf("Cb2-%d", ai), AuthorID: users[lu-1].ID },
			},
		}
		if withId {
			articleModel.ID = uint(ai)
		}
		ret = append(ret, articleModel)
	}
	return ret
}
