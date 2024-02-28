package user

import "github.com/innovationmech/swit/internal/swit-serve/db"

func registerUser(name, email string) {
	user := NewUser(name, email)

	db.GetDB().AutoMigrate(&User{})

	db.GetDB().Create(&user)

}

func getUserByName(name string) *User {
	var user *User
	db.GetDB().First(&user, "name = ?", name)

	return user
}
