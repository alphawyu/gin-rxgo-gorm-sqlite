package repository_test

import (
	"com/realworld/ginrxgogorm/repository"
	"com/realworld/ginrxgogorm/test_data"
	"com/realworld/ginrxgogorm/util"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"gorm.io/driver/sqlite"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var _ = Describe("ArticleRepository", Ordered, func() {
	var (
		test_db  *gorm.DB
		repo     *repository.ArticleRepositoryImpl
		userRepo *repository.UsersRepositoryImpl
		err      error
	)

	BeforeAll(func() {
		// Skip("generic")
		test_db, err = gorm.Open(sqlite.Open(test_data.TEST_DB_FILE), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Error),
		})
		Expect(err).NotTo(HaveOccurred())
		sql_db, err := test_db.DB()
		Expect(err).NotTo(HaveOccurred())
		sql_db.SetMaxIdleConns(3)

		test_db.AutoMigrate(&repository.UserModel{})
		test_db.AutoMigrate(&repository.ArticleModel{})
		test_db.AutoMigrate(&repository.TagModel{})
		test_db.AutoMigrate(&repository.FavoriteModel{})
		test_db.AutoMigrate(&repository.ArticleUserModel{})
		test_db.AutoMigrate(&repository.CommentModel{})

		userRepo = repository.NewUsersRepository(test_db)
		repo = repository.NewArticleRepository(userRepo, test_db)
	})

	AfterAll(func() {
		if test_data.TEST_DB_FILE != "" {
			err = os.Remove(test_data.TEST_DB_FILE)
			Expect(err).NotTo(HaveOccurred())
		}
	})

	Context("Test Article", func() {
		var (
			testUsers    []repository.UserModel
			testArticles []repository.ArticleModel
		)

		It("create users for article", func() {
			testUsers = test_data.UserModelMocker(test_db, 2)

			testArticles = test_data.GenerateTestArticles(10, 5, false, testUsers)

			for i := range testArticles {
				uam := repo.GetArticleUserModel(testArticles[i].Author.UserModel)
				testArticles[i].Author = uam

				repo.SaveOne(&testArticles[i])
			}
		})

		It("find one article", func() {
			refA1 := testArticles[1]
			a1, err := repo.FindOneArticle(&repository.ArticleModel{Slug: refA1.Slug})
			Expect(err).To(BeNil())
			Expect(a1.Slug).To(Equal(refA1.Slug))
			Expect(a1.Description).To(Equal(refA1.Description))
			Expect(a1.Author.UserModel.ID).ToNot(Equal(0))
			Expect(len(a1.Tags)).To(Equal(2))

			testArticles[1] = a1
		})

		Context("test tags ...", func() {
			It("update the tags for article#1", func() {
				a1 := testArticles[1]
				newTags := []string{
					"Tag1-12", // overlaps existing
					"Tag3",    // new
				}

				err := repo.SetTags(&a1, newTags)
				Expect(err).To(BeNil())
				Expect(len(a1.Tags)).To(Equal(2)) // only include the tags in the list

				testArticles[1] = a1
			})

			It("update the article#1", func() {
				a1 := testArticles[1]
				a1.Slug = util.UserUpdateIfNotEmpty(a1.Slug, "")
				a1.Title = util.UserUpdateIfNotEmpty(a1.Title, "T1U")
				a1.Description = util.UserUpdateIfNotEmpty(a1.Description, "")
				a1.Body = util.UserUpdateIfNotEmpty(a1.Body, "")

				err = repo.Update(&a1) // until now, the new tags are bind to the article
				Expect(err).To(BeNil())

				testArticles[1] = a1
			})

			It("find one article by the new title", func() {
				refA1 := testArticles[1]
				aX, err := repo.FindOneArticle(&repository.ArticleModel{Title: "T1U"})
				Expect(err).To(BeNil())
				Expect(aX.Slug).To(Equal(refA1.Slug))
				Expect(aX.Description).To(Equal(refA1.Description))
				Expect(aX.Author.UserModel.ID).ToNot(Equal(0))
				Expect(len(aX.Tags)).To(Equal(3)) // the existing tags are also included here
			})

			It("update article#0 for tag3", func() {
				refA0 := testArticles[0]
				a0, err := repo.FindOneArticle(&repository.ArticleModel{Slug: refA0.Slug})
				Expect(err).To(BeNil())

				newTags := []string{
					"Tag3", // new
				}

				err = repo.SetTags(&a0, newTags)
				Expect(err).To(BeNil())

				err = repo.Update(&a0) // only change the tags, now there are 2 article with tag=Tag3
				Expect(err).To(BeNil())

				testArticles[0] = a0
			})

			It("update article#0 for tag3", func() {
				tags, err := repo.GetAllTags()
				Expect(err).To(BeNil())
				Expect(len(tags)).To(Equal(11))
			})
		})

		Context("test comment ...", func() {
			It("add comments to article#0", func() {
				refA0 := testArticles[0]
				a0, err := repo.FindOneArticle(&repository.ArticleModel{Slug: refA0.Slug})
				Expect(err).To(BeNil())

				newComment := repository.CommentModel{
					Article:   a0,
					ArticleID: a0.ID,
					Author:    a0.Author,
					AuthorID:  a0.AuthorID,
					Body:      "a1-cb1",
				}

				err = repo.SaveOne(&newComment)
				Expect(err).To(BeNil())

				newComment.Model = gorm.Model{}
				newComment.Body = "a1-cb2"

				err = repo.SaveOne(&newComment)
				Expect(err).To(BeNil())

				newComment.Model = gorm.Model{}
				newComment.Body = "a1-cb3"

				err = repo.SaveOne(&newComment)
				Expect(err).To(BeNil())
			})

			It("get comments of article#0", func() {
				refA0 := testArticles[0]
				a0 := repository.ArticleModel{Slug: refA0.Slug}
				Expect(err).To(BeNil())

				err = repo.GetComments(&a0)
				Expect(err).To(BeNil())
				Expect(len(a0.Comments)).To(Equal(5))            // 2 preloaded on article creation + 3 just added
				Expect(a0.Comments[0].Author.ID).ToNot(Equal(0)) // author is loaded
				Expect(a0.Comments[4].Author.ID).ToNot(Equal(0)) // author is loaded

				testArticles[0].Comments = a0.Comments // save comments for the deletion test
			})

			It("delete comment of article#0", func() {
				refA0 := testArticles[0]

				err = repo.Delete(&refA0.Comments[0])
				Expect(err).To(BeNil())
				err = repo.Delete(&refA0.Comments[4])
				Expect(err).To(BeNil())

				a0 := repository.ArticleModel{Slug: refA0.Slug}
				err = repo.GetComments(&a0)
				Expect(err).To(BeNil())
				Expect(len(a0.Comments)).To(Equal(3)) // 2 preloaded on article creation + 3 just added - 2 just deleted
			})
		})

		Context("test favorites ...", func() {
			It("all users favorite article#0", func() {
				refA0 := testArticles[0]
				a0, err := repo.FindOneArticle(&repository.ArticleModel{Slug: refA0.Slug})
				Expect(err).To(BeNil())

				for _, u := range testUsers {
					aum := repo.GetArticleUserModel(u)
					err = repo.FavoriteBy(a0, aum)
					Expect(err).To(BeNil())
				}
			})

			It("count favorites", func() {
				a, err := repo.FindOneArticle(&repository.ArticleModel{Slug: testArticles[0].Slug})
				Expect(err).To(BeNil())

				// previous step makes all the user favorite the article
				cnt := repo.FavoritesCount(a)
				Expect(cnt).To(Equal(uint(len(testUsers))))

				a, err = repo.FindOneArticle(&repository.ArticleModel{Slug: testArticles[1].Slug})
				Expect(err).To(BeNil())

				cnt = repo.FavoritesCount(a)
				Expect(cnt).To(Equal(uint(0)))
			})

			It("is favorited", func() {
				aum := repo.GetArticleUserModel(testUsers[0])

				a, err := repo.FindOneArticle(&repository.ArticleModel{Slug: testArticles[0].Slug})
				Expect(err).To(BeNil())

				// previous step makes all the user favorite the article
				isF := repo.IsFavoriteBy(a, aum)
				Expect(isF).To(BeTrue())

				a, err = repo.FindOneArticle(&repository.ArticleModel{Slug: testArticles[1].Slug})
				Expect(err).To(BeNil())

				isF = repo.IsFavoriteBy(a, aum)
				Expect(isF).To(BeFalse())
			})

			It("user#0 unfavorite article#0", func() {
				refA0 := testArticles[0]
				a0, err := repo.FindOneArticle(&repository.ArticleModel{Slug: refA0.Slug})
				Expect(err).To(BeNil())

				aum := repo.GetArticleUserModel(testUsers[0])
				err = repo.UnfavoriteBy(a0, aum)
				Expect(err).To(BeNil())

				isF := repo.IsFavoriteBy(a0, aum)
				Expect(isF).To(BeFalse())

				cnt := repo.FavoritesCount(a0)
				Expect(cnt).To(Equal(uint(len(testUsers) - 1)))
			})
		})

		DescribeTable("findManyArticle by ...",
			func(tag, author, limit, offset, favorited string, oCnt, oLen int) {
				aAry, cnt, err := repo.FindManyArticle(tag, author, limit, offset, favorited)
				Expect(err).To(BeNil())
				Expect(cnt).To(Equal(oCnt), "count ...")
				Expect(len(aAry)).To(Equal(oLen), "returned list ...")
			},
			Entry("tag=Tag3", "Tag3", "", "", "", "", 2, 2),
			Entry("author=user1", "", "user1", "", "", "", 2, 2),
			Entry("favoriteBy=user2", "", "", "", "", "user2", 1, 1),
			Entry("no restrict", "", "", "", "", "", 5, 5),
			Entry("limit 2", "", "", "2", "", "", 5, 2),
			Entry("offset 2 (remain 3)", "", "", "", "2", "", 5, 3),
			Entry("author=user1, limit 1", "", "user1", "1", "", "", 2, 1),
			Entry("author=user1, offset 1", "", "user1", "", "1", "", 2, 1),
			Entry("author=user1, offset 1, limit1", "", "user1", "1", "1", "", 2, 1), // offset first ...
			Entry("author=user1, offset 2", "", "user1", "", "2", "", 2, 0),
			Entry("favoriteBy=user2, offset 1", "", "", "", "1", "user2", 1, 0),
			Entry("tag overrides others", "Tag3", "", "", "", "user2", 2, 2),
		)

		Context("test feed (following) ...", func() {
			var testAum repository.ArticleUserModel
			It("user2 follows user1", func() {
				testUser := test_data.UserModelMocker(test_db, 1)[0]

				for i, _ := range testUsers {
					err := userRepo.Following(testUser, testUsers[i])
					Expect(err).To(BeNil())
				}

				testAum = repo.GetArticleUserModel(testUser)
			})

			DescribeTable("GetArticleFeed by tag",
				func(offset, limit string, oCnt, oLen int) {
					aAry, cnt, err := repo.GetArticleFeed(testAum, limit, offset)
					Expect(err).To(BeNil())
					Expect(cnt).To(Equal(oCnt), "count ...")
					Expect(len(aAry)).To(Equal(oLen), "returned list ...")
				},
				Entry("unrestricted", "", "", 5, 5),
				Entry("limit 3", "", "3", 5, 3),
				Entry("offset 1, limit 3", "1", "3", 5, 3),
				Entry("offset 3, limit 3", "3", "3", 5, 2),
				Entry("limit 0", "", "0", 5, 0),
			)

			It("GatherLoginUserStat", func() {
				refA0 := testArticles[0]
				a0, err := repo.FindOneArticle(&repository.ArticleModel{Slug: refA0.Slug})
				Expect(err).To(BeNil())

				followingFlag, favorite, favoriteCount, _ := repo.GatherLoginUserStat(testAum, a0)
				Expect(followingFlag).To(BeTrue())
				Expect(favorite).To(BeFalse(), "ux favorite a0...")
				Expect(favoriteCount).To(Equal(uint(1)))

				testAum = repo.GetArticleUserModel(testUsers[1])

				followingFlag, favorite, favoriteCount, _ = repo.GatherLoginUserStat(testAum, a0)
				Expect(followingFlag).To(BeFalse())
				Expect(favorite).To(BeTrue(), "u1 favorite a0...")
				Expect(favoriteCount).To(Equal(uint(1)))
			})
		})

		It("delete article", func() {
			refA1 := testArticles[1]
			err := repo.Delete(&repository.ArticleModel{Slug: refA1.Slug})
			Expect(err).To(BeNil())

			_, err = repo.FindOneArticle(&repository.ArticleModel{Slug: refA1.Slug})
			Expect(err).ToNot(BeNil())

			_, cnt, err := repo.FindManyArticle("", "", "", "", "")
			Expect(err).To(BeNil())
			Expect(cnt).To(Equal(4))
		})

	})
})
