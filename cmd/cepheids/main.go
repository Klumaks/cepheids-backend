package main

import (
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"

	"cepheids-backend/internal/app/config"
	"cepheids-backend/internal/app/dsn"
	"cepheids-backend/internal/app/handler"
	"cepheids-backend/internal/app/repository"
	"cepheids-backend/internal/pkg"
)

func main() {
	_ = godotenv.Load()

	router := gin.Default()

	conf, err := config.NewConfig()
	if err != nil {
		logrus.Fatalf("error loading config: %v", err)
	}

	rep, err := repository.New(dsn.FromEnv())
	if err != nil {
		logrus.Fatalf("error initializing repository: %v", err)
	}

	hand := handler.NewHandler(rep)
	app := pkg.NewApp(conf, router, hand)
	app.RunApp()
}
