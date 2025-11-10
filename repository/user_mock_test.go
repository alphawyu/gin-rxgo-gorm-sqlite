package repository_test

import (
	"com/realworld/ginrxgogorm/repository"
	"database/sql"
	"database/sql/driver"
	"errors"

	"time"

	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"gorm.io/driver/sqlite"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type AnyTime struct {
}

// Match satisfies sqlmock.Argument interface
func (a AnyTime) Match(v driver.Value) bool {
	if v == nil {
		return false
	}
	if _, ok := v.(time.Time); ok {
		return true
	}
	_, ok := v.(sql.NullTime)
	return ok
}

type AnyNil struct {
}

// Match satisfies sqlmock.Argument interface
func (a AnyNil) Match(v driver.Value) bool {
	return v == nil
}

var _ = Describe("UserRepository", func() {
	var (
		db   *gorm.DB
		mock sqlmock.Sqlmock
		repo *repository.UsersRepositoryImpl
	)

	BeforeEach(func() {
		// Skip("generic")
		var err error
		var sqlDB *sql.DB
		var matcher = sqlmock.QueryMatcherRegexp
		// var matcher = sqlmock.QueryMatcherEqual
		sqlDB, mock, err = sqlmock.New(sqlmock.QueryMatcherOption(matcher))
		Expect(err).NotTo(HaveOccurred())

		dsn := "sqlmock_db_0"
		// NOTE: enable this when test with "sqlite"
		//     - also need remove all the WithArg() calls - sqlite query does not limit 1
		mock.ExpectQuery("select sqlite_version()").
			WillReturnRows(sqlmock.NewRows([]string{"sqlite_version()"}).AddRow("3.39.5"))
		dbDialector := sqlite.New(
			sqlite.Config{
				DSN:        dsn,
				Conn:       sqlDB,
				DriverName: "sqlite",
				// PreferSimpleProtocol: true, // disables implicit prepared statement usage
			})

		db, err = gorm.Open(
			dbDialector,
			&gorm.Config{
				Logger: logger.Default.LogMode(logger.Info),
			},
		)
		Expect(err).NotTo(HaveOccurred())

		repo = repository.NewUsersRepository(db)
	})

	AfterEach(func() {
		err := mock.ExpectationsWereMet()
		Expect(err).NotTo(HaveOccurred())
	})

	Context("FindOneUser()", func() {
		var rows *sqlmock.Rows
		expected := repository.UserModel{
			Model: gorm.Model{ID: uint(10)},
			Username:     "username10",
			Email:        "t10@t.t",
			Bio:          "b10",
			Image:        nil,
			PasswordHash: "",
		}

		BeforeEach(func() {
			rows = sqlmock.NewRows([]string{"id", "username", "email", "bio"}).
				FromCSVString("10,username10,t10@t.t,b10").
				FromCSVString("11,username11,t11@t.t,b11")

		})
		It("FindOneUserById is a pass through call", func() {
			mock.ExpectQuery("SELECT (.*) FROM .user_models. WHERE (.*)").
				WithArgs(10).
				WillReturnRows(rows)

			r, err := repo.FindOneUserById(10)
			Expect(err).To(BeNil())
			Expect((r)).To(Equal(expected))
		})

		It("should successfully process first record", func() {
			mock.ExpectQuery("SELECT (.*) FROM .user_models. WHERE (.*)").
				WithArgs("username0").
				WillReturnRows(rows)

			r, err := repo.FindOneUser(repository.UserModel{Username: "username0"})
			Expect(err).To(BeNil())
			Expect((r)).To(Equal(expected))
		})

		It("should properly handle error", func() {
			mock.ExpectQuery("SELECT (.*) FROM .user_models. WHERE (.*)").
				WithArgs("username0").
				WillReturnError(gorm.ErrRecordNotFound)

			r, err := repo.FindOneUser(repository.UserModel{Username: "username0"})
			Expect(err).To(Equal(gorm.ErrRecordNotFound))
			Expect(r).To(Equal(repository.UserModel{}))
		})
	})

	Context("SaveOne()", func() {
		testUser := repository.UserModel{
			Username:     "username10",
			Email:        "test@email.com",
			Bio:          "b10",
			Image:        nil,
			PasswordHash: "",
		}
		insertRtn := sqlmock.NewRows([]string{"id"}).AddRow(1)
		insertErr := gorm.ErrModelAccessibleFieldsRequired

		It("should successfully insert record when no match is found for name and format", func() {
			mock.ExpectBegin()
			mock.ExpectQuery("INSERT INTO .user_models. (.*)").
				WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), "username10", "test@email.com", "b10", nil, sqlmock.AnyArg()).
				WillReturnRows(insertRtn)
			mock.ExpectCommit()

			_, err := repo.SaveOne(testUser)
			Expect(err).To(BeNil())
		})

		It("should rollback when insert record returns error", func() {
			mock.ExpectBegin()
			mock.ExpectQuery("INSERT INTO .user_models. (.*)").
				WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), "username10", "test@email.com", "b10", nil, sqlmock.AnyArg()).
				WillReturnError(insertErr)
			mock.ExpectRollback()

			_, err := repo.SaveOne(testUser)
			Expect(err).To(Equal(insertErr))
		})
	})

	Context("Update()", func() {
		mi := repository.UserModel{
			Username: "username0",
			Email:    "test@email.com",
			Bio:      "someBio",
		}

		It("should successfully update record when match is found for name and format", func() {
			mock.ExpectBegin()
			mock.ExpectExec("UPDATE .user_models.(.*)").
				WithArgs(sqlmock.AnyArg(), "username0", "test@email.com", "someBio", 1).
				WillReturnResult(sqlmock.NewResult(0, 1)) // no insert id, 1 affected row
			mock.ExpectCommit()

			_, err := repo.Update(repository.UserModel{Model: gorm.Model{ID: 1}}, mi)
			Expect(err).To(BeNil())
		})

		It("should rollback when update record returns error", func() {
			updateErr := errors.New("some error")

			mock.ExpectBegin()
			mock.ExpectExec("UPDATE .user_models. (.*)").
				WithArgs(sqlmock.AnyArg(), "username0", "test@email.com", "someBio", 1).
				WillReturnError(updateErr)
			mock.ExpectRollback()

			_, err := repo.Update(repository.UserModel{Model: gorm.Model{ID: 1}}, mi)
			Expect(err).To(Equal(updateErr))
		})
	})

	Context("GetFollowings()", func() {
		var rows *sqlmock.Rows
		expected := []repository.UserModel{
			{	Model:        gorm.Model{ID: uint(10)},
				Username:     "username10",
				Email:        "t10@t.com",
				Bio:          "b10",
				Image:        nil,
				PasswordHash: "",
			},
			{	Model:        gorm.Model{ID: uint(11)},
				Username:     "username11",
				Email:        "t11@t.com",
				Bio:          "b11",
				Image:        nil,
				PasswordHash: "",
			},
		}

		BeforeEach(func() {
			rows = sqlmock.NewRows([]string{"id", "username", "email", "bio"}).
				FromCSVString("10,username10,t10@t.com,b10").
				FromCSVString("11,username11,t11@t.com,b11")

		})

		It("should the following users to the user (id = 1)", func() {
			mock.ExpectQuery("SELECT (.*) FROM .user_models. JOIN .user_model_followings. ON (.*)").
				WithArgs(1).
				WillReturnRows(rows)

			r, err := repo.GetFollowings(repository.UserModel{Model: gorm.Model{ID: 1}})
			Expect(err).To(BeNil())
			Expect(r).To(Equal(expected))
		})

		It("should return error with nil ", func() {
			mock.ExpectQuery("SELECT (.*) FROM .user_models. JOIN .user_model_followings. ON (.*)").
				WithArgs(1).
				WillReturnError(gorm.ErrRecordNotFound)

			_, err := repo.GetFollowings(repository.UserModel{Model: gorm.Model{ID: 1}})
			Expect(err).To(Equal(gorm.ErrRecordNotFound))
		})
	})

	Context("IsFollowings()", func() {
		var rows *sqlmock.Rows

		BeforeEach(func() {
			rows = sqlmock.NewRows([]string{"id", "username", "email", "bio"}).
				FromCSVString("10,username10,t10@t.com,b10").
				FromCSVString("11,username11,t11@t.com,b11")

		})

		It("should the following users to the user (id = 1)", func() {
			mock.ExpectQuery("SELECT (.*) FROM .user_models. JOIN .user_model_followings. ON (.*)").
				WithArgs(1, 2).
				WillReturnRows(rows)

			r, err := repo.IsFollowings(repository.UserModel{Model: gorm.Model{ID: 1}}, repository.UserModel{Model: gorm.Model{ID: 2}})
			Expect(err).To(BeNil())
			Expect(r).To(BeTrue())
		})

		It("should return error with nil ", func() {
			mock.ExpectQuery("SELECT (.*) FROM .user_models. JOIN .user_model_followings. ON (.*)").
				WithArgs(1, 2).
				WillReturnError(gorm.ErrRecordNotFound)

			r, err := repo.IsFollowings(repository.UserModel{Model: gorm.Model{ID: 1}}, repository.UserModel{Model: gorm.Model{ID: 2}})
			Expect(err).To(Equal(gorm.ErrRecordNotFound))
			Expect(r).To(BeFalse())
		})
	})
})
