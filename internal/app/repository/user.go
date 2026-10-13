package repository

import (
	"cepheids-backend/internal/app/ds"
	"golang.org/x/crypto/bcrypt"
)

func (r *Repository) CreateUser(login, password string) (ds.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return ds.User{}, err
	}
	user := ds.User{Login: login, Password: string(hash)}
	err = r.db.Create(&user).Error
	return user, err
}
func (r *Repository) GetUserByLogin(login string) (ds.User, error) {
	var user ds.User
	err := r.db.Where("login = ?", login).First(&user).Error
	return user, err
}
