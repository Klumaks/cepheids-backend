package main

import (
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"cepheids-backend/internal/app/ds"
	"cepheids-backend/internal/app/dsn"
)

func main() {
	_ = godotenv.Load()

	db, err := gorm.Open(postgres.Open(dsn.FromEnv()), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// Порядок важен: сначала users, потом spectral_classes, потом likes
	err = db.AutoMigrate(
		&ds.User{},
		&ds.SpectralClass{},
		&ds.Like{},
	)
	if err != nil {
		panic("cant migrate db")
	}
}
