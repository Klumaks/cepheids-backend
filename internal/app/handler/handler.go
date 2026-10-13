package handler

import (
	"cepheids-backend/internal/app/auth"
	"cepheids-backend/internal/app/repository"
	"cepheids-backend/internal/pkg"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	Repository    *repository.Repository
	MinioClient   *pkg.MinioClient
	CepheidTokens *auth.CepheidTokens
}

func NewHandler(r *repository.Repository, minioClient *pkg.MinioClient, tokens *auth.CepheidTokens) *Handler {
	return &Handler{Repository: r, MinioClient: minioClient, CepheidTokens: tokens}
}
func (h *Handler) RegisterHandler(router *gin.Engine) {
	api := router.Group("/api")
	api.GET("/cepheids", h.CepheidAuthorization(false), h.GetCepheids)
	api.GET("/cepheids/feed", h.CepheidAuthorization(false), h.GetFeed)
	api.GET("/cepheids/feed/:id", h.CepheidAuthorization(false), h.GetFeedByID)
	api.GET("/cepheids/draft", h.CepheidAuthorization(true), h.GetDraft)
	api.POST("/cepheids", h.CepheidAuthorization(true), h.CreateCepheid)
	api.PUT("/cepheids/:id/publish", h.CepheidAuthorization(true), h.PublishCepheid)
	api.DELETE("/cepheids/:id", h.CepheidAuthorization(true), h.DeleteCepheid)
	api.POST("/cepheids/:id/like", h.CepheidAuthorization(true), h.LikeCepheid)
	api.POST("/users/register", h.RegisterUser)
	api.POST("/users/login", h.LoginUser)
	api.POST("/users/logout", h.CepheidAuthorization(true), h.LogoutUser)
}
