package api

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"cepheids-backend/internal/app/handler"
	"cepheids-backend/internal/app/repository"
	"cepheids-backend/internal/pkg"
)

func StartServer() {
	log.Println("Starting server")

	// Инициализация репозитория
	repo, err := repository.New(os.Getenv("DATABASE_URL"))
	if err != nil {
		logrus.Fatal("ошибка инициализации репозитория: ", err)
	}

	// Инициализация Minio клиента
	minioClient, err := pkg.NewMinioClient(
		"localhost:9000",
		"minioadmin",
		"minioadmin",
		"cepheids",
	)
	if err != nil {
		logrus.Fatal("ошибка инициализации Minio: ", err)
	}

	// Инициализация handler
	h := handler.NewHandler(repo, minioClient)

	// Настройка роутера
	r := gin.Default()

	// Регистрация маршрутов
	h.RegisterHandler(r)

	// Запуск сервера
	r.Run(":8080")
	log.Println("Server down")
}
