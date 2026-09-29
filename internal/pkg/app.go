package pkg

import (
	"net"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"cepheids-backend/internal/app/config"
	"cepheids-backend/internal/app/handler"
)

type Application struct {
	Config  *config.Config
	Router  *gin.Engine
	Handler *handler.Handler
}

func NewApp(c *config.Config, r *gin.Engine, h *handler.Handler) *Application {
	return &Application{
		Config:  c,
		Router:  r,
		Handler: h,
	}
}

func (a *Application) RunApp() {
	logrus.Info("Server start up")

	a.Handler.RegisterHandler(a.Router)
	a.Handler.RegisterStatic(a.Router)

	// Явный IPv4-only сокет: WSL пробросит его в Windows как 127.0.0.1
	addr := "0.0.0.0:8080"
	listener, err := net.Listen("tcp4", addr)
	if err != nil {
		logrus.Fatalf("listen error: %v", err)
	}

	logrus.Info("Listening on ", addr)
	if err := a.Router.RunListener(listener); err != nil {
		logrus.Fatal(err)
	}
	logrus.Info("Server down")
}
