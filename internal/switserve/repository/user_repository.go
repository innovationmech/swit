package repository

import (
	"github.com/innovationmech/swit/internal/switserve/model"
	"gorm.io/gorm"
)

// UserRepository is the interface for the user repository.
type UserRepository interface {
	CreateUser(user *model.User) error
	GetUserByUsername(username string) (*model.User, error)
	GetUserByEmail(email string) (*model.User, error)
	UpdateUser(user *model.User) error
	DeleteUser(id string) error
}

// userRepository is the implementation of the UserRepository interface.
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new user repository.
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

// CreateUser creates a new user.
func (u userRepository) CreateUser(user *model.User) error {
	result := u.db.Create(user)
	return result.Error
}

// GetUserByUsername gets a user by username.
func (u userRepository) GetUserByUsername(username string) (*model.User, error) {
	var user model.User
	result := u.db.Where("username = ?", username).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

// GetUserByEmail gets a user by email.
func (u userRepository) GetUserByEmail(email string) (*model.User, error) {
	var user model.User
	result := u.db.Where("email = ?", email).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

// UpdateUser updates a user.
func (u userRepository) UpdateUser(user *model.User) error {
	result := u.db.Save(user)
	return result.Error
}

// DeleteUser deletes a user.
func (u userRepository) DeleteUser(id string) error {
	result := u.db.Delete(&model.User{}, id)
	return result.Error
}
