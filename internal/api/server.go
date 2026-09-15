package api

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"cepheids-backend/internal/app/handler"
	"cepheids-backend/internal/app/repository"
)

func StartServer() {
	log.Println("Starting server")

	repo, err := repository.NewRepository()
	if err != nil {
		logrus.Error("ошибка инициализации репозитория")
	}

	h := handler.NewHandler(repo)

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")
	r.Static("/static", "./resources")

	// Маршруты
	r.GET("/feed", h.Feed)
	r.GET("/feed/:id", h.Feed)
	r.GET("/add", h.Add)
	r.GET("/classes", h.Classes)

	// Редирект с корня на ленту
	r.GET("/", func(ctx *gin.Context) {
		ctx.Redirect(302, "/feed")
	})

	r.Run(":8080")
	log.Println("Server down")
}
