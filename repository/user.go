package repository

import (
	log "github.com/sirupsen/logrus"

	"gorm.io/gorm"
)

type UserModel struct {
	gorm.Model
	Username     string      `gorm:"column:username"`
	Email        string      `gorm:"column:email;unique_index"`
	Bio          string      `gorm:"column:bio;size:1024"`
	Image        *string     `gorm:"column:image"`
	PasswordHash string      `gorm:"column:password;not null"`
	Followings   []UserModel `gorm:"many2many:user_model_followings"`
}

//go:generate mockgen -destination mock/users_repo_mock.go -source user.go UsersRepository
type UsersRepository interface {
	FindOneUserById(id uint) (UserModel, error)
	FindOneUser(refUser UserModel) (UserModel, error)
	SaveOne(data UserModel) (UserModel, error)
	Update(model, data UserModel) (UserModel, error)

	Following(u, v UserModel) error
	Unfollowing(u, v UserModel) error
	IsFollowings(u, v UserModel) (bool, error)
	GetFollowings(u UserModel) ([]UserModel, error)
}

type UsersRepositoryImpl struct {
	db *gorm.DB
}

func NewUsersRepository(db *gorm.DB) *UsersRepositoryImpl {
	return &UsersRepositoryImpl{
		db,
	}
}

// to ensure the __impl implements the interface at compile-time
var _ UsersRepository = &UsersRepositoryImpl{}

func (repo *UsersRepositoryImpl) FindOneUserById(id uint) (UserModel, error) {
	return repo.FindOneUser(UserModel{Model: gorm.Model{ID: id}})
}

func (repo *UsersRepositoryImpl) FindOneUser(refUser UserModel) (UserModel, error) {
	var model UserModel
	err := repo.db.Where(refUser).First(&model).Error
	if err != nil {
		log.Errorf("FindOneUser (gorm:Where,First) error %v", err)
	}
	return model, err
}

func (repo *UsersRepositoryImpl) SaveOne(data UserModel) (UserModel, error) {
	err := repo.db.Save(&data).Error
	if err != nil {
		log.Errorf("SaveOne (gorm:Save) error %v", err)
	}
	return data, err
}

func (repo *UsersRepositoryImpl) Update(model, data UserModel) (UserModel, error) {
	err := repo.db.Model(&model).Updates(&data).Error
	if err != nil {
		log.Errorf("Update (gorm:Model,Updates) error %v", err)
	}
	return data, err
}

func (r *UsersRepositoryImpl) Following(u, v UserModel) error {
	err := r.db.Model(&u).Association("Followings").Append(&v)
	if err != nil {
		log.Errorf("Following (gorm:Model,Association,Append) error %v", err)
	}
	return err
}

func (r *UsersRepositoryImpl) IsFollowings(u, v UserModel) (bool, error) {
	var followings []UserModel
	if err := r.db.Model(&u).Association("Followings").Find(&followings, "id=?", v.ID); err != nil {
		log.Errorf("IsFollowings (gorm:Model,Association,Find) error %v", err)
		return false, err
	}
	return len(followings) > 0, nil
}

func (r *UsersRepositoryImpl) Unfollowing(u, v UserModel) error {
	err := r.db.Model(&u).Association("Followings").Delete(&v)
	if err != nil {
		log.Errorf("Unfollowing (gorm:Model,Association,Delete) error %v", err)
	}
	return err
}

func (r *UsersRepositoryImpl) GetFollowings(u UserModel) ([]UserModel, error) {
	var followings []UserModel
	if err := r.db.Model(&u).Association("Followings").Find(&followings); err != nil {
		log.Errorf("GetFollowings (gorm:Model,Association,Find) error %v", err)
		return followings, err
	}

	return followings, nil
}
