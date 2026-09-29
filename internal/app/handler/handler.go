package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"cepheids-backend/internal/app/repository"
	"cepheids-backend/internal/pkg"
)

const minioURL = "http://localhost:9000/cepheids"

var currentUserID uint = 1

func GetCurrentUserID() uint {
	return currentUserID
}

type Handler struct {
	Repository  *repository.Repository
	MinioClient *pkg.MinioClient
}

func NewHandler(r *repository.Repository, minioClient *pkg.MinioClient) *Handler {
	return &Handler{
		Repository:  r,
		MinioClient: minioClient,
	}
}

func (h *Handler) RegisterHandler(router *gin.Engine) {
	api := router.Group("/api")
	{
		// Домен услуги
		api.GET("/cepheids", h.GetCepheids)
		api.GET("/cepheids/feed", h.GetFeed)
		api.GET("/cepheids/feed/:id", h.GetFeedByID)
		api.GET("/cepheids/draft", h.GetDraft)
		api.POST("/cepheids", h.CreateCepheid)
		api.PUT("/cepheids/:id/publish", h.PublishCepheid)
		api.DELETE("/cepheids/:id", h.DeleteCepheid)
		api.POST("/cepheids/:id/like", h.LikeCepheid)

		// Домен пользователь
		api.POST("/users/register", h.RegisterUser)
		api.POST("/users/login", h.LoginUser)
		api.POST("/users/logout", h.LogoutUser)
	}
}

func (h *Handler) errorHandler(ctx *gin.Context, code int, err error) {
	logrus.Error(err.Error())
	ctx.Status(code)
}
