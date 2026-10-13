package main

import (
	"log"

	"cepheids-backend/internal/app/ds"
	"cepheids-backend/internal/app/dsn"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func hashCepheidPasswords(db *gorm.DB) (int, error) {
	count := 0
	err := db.Transaction(func(tx *gorm.DB) error {
		var observers []ds.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Find(&observers).Error; err != nil {
			return err
		}
		for _, observer := range observers {
			if _, err := bcrypt.Cost([]byte(observer.Password)); err == nil {
				continue
			}
			hash, err := bcrypt.GenerateFromPassword([]byte(observer.Password), bcrypt.DefaultCost)
			if err != nil {
				return err
			}
			if err := tx.Model(&ds.User{}).Where("id = ?", observer.ID).Update("password", string(hash)).Error; err != nil {
				return err
			}
			count++
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return count, nil
}
func main() {
	_ = godotenv.Load()
	db, err := gorm.Open(postgres.Open(dsn.FromEnv()), &gorm.Config{})
	if err != nil {
		log.Fatal("cannot connect database")
	}
	count, err := hashCepheidPasswords(db)
	if err != nil {
		log.Fatal("password migration rolled back: ", err)
	}
	log.Printf("Hashed passwords for %d observers; existing hashes unchanged", count)
}
