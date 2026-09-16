package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"cepheids-backend/internal/app/repository"
)

const (
	minioURL = "http://localhost:9000/cepheids"
	// TODO: заменить на реальную авторизацию в ЛР4
	currentUserID = 1
)

type Handler struct {
	Repository *repository.Repository
}

func NewHandler(r *repository.Repository) *Handler {
	return &Handler{Repository: r}
}

func (h *Handler) RegisterHandler(router *gin.Engine) {
	router.GET("/feed", h.Feed)
	router.GET("/feed/:id", h.Feed)
	router.GET("/add", h.Add)
	router.GET("/classes", h.Classes)
	router.POST("/add", h.CreateDraft)
	router.POST("/publish", h.Publish)
	router.POST("/classes/delete", h.DeleteClass)
}

func (h *Handler) RegisterStatic(router *gin.Engine) {
	router.LoadHTMLGlob("templates/*")
	router.Static("/static", "./resources")
}

func (h *Handler) errorHandler(ctx *gin.Context, code int, err error) {
	logrus.Error(err.Error())
	ctx.JSON(code, gin.H{
		"status":      "error",
		"description": err.Error(),
	})
}
