package repository

import "cepheids-backend/internal/app/ds"

// CreateUser — создание пользователя
func (r *Repository) CreateUser(login, password string) (ds.User, error) {
	user := ds.User{
		Login:    login,
		Password: password, // В ЛР4 добавишь хэширование
	}
	err := r.db.Create(&user).Error
	return user, err
}
