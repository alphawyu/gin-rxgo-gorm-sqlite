package repository_test

import (
	"com/realworld/ginrxgogorm/repository"
	"com/realworld/ginrxgogorm/test_data"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"gorm.io/driver/sqlite"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var _ = Describe("UserRepository", Ordered, func() {
	var (
		repo    *repository.UsersRepositoryImpl
		test_db *gorm.DB
		err     error
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

		repo = repository.NewUsersRepository(test_db)
	})

	AfterAll(func() {
		if test_data.TEST_DB_FILE != "" {
			err = os.Remove(test_data.TEST_DB_FILE)
			Expect(err).NotTo(HaveOccurred())
		}
	})

	Context("Follow & Followed", func() {
		var a, b, c repository.UserModel
		It("init - no followings", func() {
			users := test_data.UserModelMocker(test_db, 3)
			a = users[0]
			b = users[1]
			c = users[2]

			followings, err := repo.GetFollowings(a)
			Expect(err).To(BeNil())
			Expect(len(followings)).To(Equal(0), "expect zero followings on creation")
			Expect(repo.IsFollowings(a, b)).To(BeFalse(), "expect no following on creation")
		})

		It("b follows a", func() {
			Expect(repo.Following(a, b)).NotTo(HaveOccurred())
			followings, err := repo.GetFollowings(a)
			Expect(err).To(BeNil())
			Expect(len(followings)).To(Equal(1), "expect one following of b")
			Expect(followings[0].ID).To(Equal(b.ID), "the following user id equals to b")
			Expect(repo.IsFollowings(a, b)).Should(BeTrue(), "b is following a")
		})

		It("b, c follows a", func() {
			Expect(repo.Following(a, c)).NotTo(HaveOccurred())
			followings, err := repo.GetFollowings(a)
			Expect(err).To(BeNil())
			Expect(len(followings)).To(Equal(2))
			Expect(followings[0].ID).To(Equal(b.ID))
			Expect(followings[1].ID).To(Equal(c.ID))
			Expect(repo.IsFollowings(a, c)).Should(BeTrue())
		})

		It("b unfollows a", func() {
			Expect(repo.Unfollowing(a, b)).NotTo(HaveOccurred())
			followings, err := repo.GetFollowings(a)
			Expect(err).To(BeNil())
			Expect(len(followings)).To(Equal(1))
			Expect(followings[0].ID).To(Equal(c.ID))
			Expect(repo.IsFollowings(a, c)).Should(BeTrue())
			Expect(repo.IsFollowings(a, b)).Should(BeFalse())
		})
	})
})
