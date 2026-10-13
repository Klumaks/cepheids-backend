package main

import (
	"cepheids-backend/internal/app/ds"
	"cepheids-backend/internal/app/dsn"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
)

func main() {
	_ = godotenv.Load()
	db, err := gorm.Open(postgres.Open(dsn.FromEnv()), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database: ", err)
	}
	if err := db.AutoMigrate(&ds.User{}, &ds.SpectralClass{}, &ds.Like{}); err != nil {
		log.Fatal(err)
	}
	// Fail on existing duplicates rather than remove any user's data.
	if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_spectral_class_observer_draft ON spectral_classes (creator_id) WHERE status = 'draft'").Error; err != nil {
		log.Fatal("cannot enforce one draft per observer; inspect existing duplicate drafts: ", err)
	}
	log.Println("Cepheid schema migrated")
}
