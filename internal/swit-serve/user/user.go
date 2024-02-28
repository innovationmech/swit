package user

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Name  string `json:"name"`
	Email string `json:"email" gorm:"type:varchar(100);unique_index"`
}

func NewUser(name, email string) *User {
	return &User{
		Name:  name,
		Email: email,
	}
}
