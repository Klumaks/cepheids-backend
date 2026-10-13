package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"

	"cepheids-backend/internal/api"
)

// @title Cepheids API
// @version 4.0
// @description Спектральные классы цефеид: калибровка зависимости период–светимость, JWT и Redis blacklist.
// @BasePath /api
// @securityDefinitions.apikey CepheidBearer
// @in header
// @name Authorization
// @description Введите Bearer и JWT из ответа /api/users/login.
func main() {
	// Загружаем переменные окружения
	_ = godotenv.Load()

	// Устанавливаем DATABASE_URL если не задан
	if os.Getenv("DATABASE_URL") == "" {
		os.Setenv("DATABASE_URL", "host=localhost user=myuser password=mypassword dbname=cepheids_db port=5433 sslmode=disable")
	}

	log.Println("Application start!")
	api.StartServer()
	log.Println("Application terminated!")
}
