package api

import (
	_ "cepheids-backend/docs"
	"cepheids-backend/internal/app/auth"
	"cepheids-backend/internal/app/handler"
	"cepheids-backend/internal/app/repository"
	"cepheids-backend/internal/pkg"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"log"
	"os"
)

func StartServer() {
	tokens, redisClient, err := auth.CepheidTokensFromEnv()
	if err != nil {
		logrus.Fatal("ошибка настройки JWT: ", err)
	}
	defer redisClient.Close()
	repo, err := repository.New(os.Getenv("DATABASE_URL"))
	if err != nil {
		logrus.Fatal("ошибка инициализации репозитория: ", err)
	}
	minioClient, err := pkg.NewMinioClient("localhost:9000", "minioadmin", "minioadmin", "cepheids")
	if err != nil {
		logrus.Fatal("ошибка инициализации MinIO: ", err)
	}
	h := handler.NewHandler(repo, minioClient, tokens)
	r := gin.Default()
	h.RegisterHandler(r)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	address := os.Getenv("CEPHEIDS_ADDR")
	if address == "" {
		address = ":8080"
	}
	log.Println("Cepheids API listening on", address)
	if err := r.Run(address); err != nil {
		logrus.Fatal(err)
	}
}
