package main

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"cepheids-backend/internal/app/ds"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestHashCepheidPasswordsIntegration(t *testing.T) {
	url := os.Getenv("CEPHEIDS_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set CEPHEIDS_TEST_DATABASE_URL to an isolated cepheids_lab4_test database")
	}
	db, err := gorm.Open(postgres.Open(url), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	var name string
	if err := db.Raw("SELECT current_database()").Scan(&name).Error; err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(name, "cepheids_lab4_test") {
		t.Fatal("isolated test database required")
	}
	if err := db.AutoMigrate(&ds.User{}); err != nil {
		t.Fatal(err)
	}
	observer := ds.User{Login: fmt.Sprintf("legacy_%x", time.Now().UnixNano()), Password: "legacy-observation"}
	if err := db.Create(&observer).Error; err != nil {
		t.Fatal(err)
	}
	defer db.Delete(&observer)
	count, err := hashCepheidPasswords(db)
	if err != nil || count < 1 {
		t.Fatalf("count=%d err=%v", count, err)
	}
	var stored ds.User
	if err := db.First(&stored, observer.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.ID != observer.ID || stored.Login != observer.Login || bcrypt.CompareHashAndPassword([]byte(stored.Password), []byte(observer.Password)) != nil {
		t.Fatal("migration changed identity or password")
	}
	firstHash := stored.Password
	count, err = hashCepheidPasswords(db)
	if err != nil || count != 0 {
		t.Fatalf("migration not idempotent: count=%d err=%v", count, err)
	}
	db.First(&stored, observer.ID)
	if stored.Password != firstHash {
		t.Fatal("existing bcrypt hash was rewritten")
	}
	t.Log("Legacy password hashed, observer ID preserved, second run unchanged")
}
