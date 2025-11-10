package middleware_test

import (
	"com/realworld/ginrxgogorm/middleware"
	"com/realworld/ginrxgogorm/repository"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gorm.io/gorm"
)

var image_url = "https://golang.org/doc/gopher/frontpage.png"

func newUserModel() repository.UserModel {
	return repository.UserModel{
		Model:        gorm.Model{ID: 2},
		Username:     "asd123!@#ASD",
		Email:        "test@t.t",
		Bio:          "gibberish",
		Image:        &image_url,
		PasswordHash: "",
	}
}

var _ = Describe("test middleware", func() {
	It("CheckPassword provided with empty password", func() {
		//Testing repository.UserModel's password feature
		userModel := newUserModel()
		err := middleware.CheckPassword(&userModel, "")
		Expect(err).ToNot(BeNil(), "empty password should return err")
	})

	It("SetPassword to empty password", func() {
		userModel := newUserModel()
		userModel.PasswordHash = "original hash"
		oum, err := middleware.SetPassword(&userModel, "")
		Expect(err).ToNot(BeNil(), "empty password can not be set")
		Expect(oum.PasswordHash).To(Equal(userModel.PasswordHash), "passwordHash is not updated")
	})

	It("SetPassword to valid password", func() {
		userModel := newUserModel()
		_, err := middleware.SetPassword(&userModel, "asd123!@#ASD")
		Expect(err).To(BeNil(), "password should be set successful")
		Expect(len(userModel.PasswordHash)).To(Equal(60), "password hash length should be 60")

		err = middleware.CheckPassword(&userModel, "sd123!@#ASD") // different password
		Expect(err).ToNot(BeNil(), "password should be checked and not validated")

		err = middleware.CheckPassword(&userModel, "asd123!@#ASD") // same password
		Expect(err).To(BeNil(), "password should be checked and validated")
	})

	It("GenToken to valid password", func() {
		token := middleware.GenToken(10)
		Expect(token).ToNot(BeNil(), "token is created")
	})

})
