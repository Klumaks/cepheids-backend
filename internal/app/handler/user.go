package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// POST /api/users/register — регистрация
func (h *Handler) RegisterUser(ctx *gin.Context) {
	var req struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	user, err := h.Repository.CreateUser(req.Login, req.Password)
	if err != nil {
		logrus.Error(err)
		ctx.Status(http.StatusConflict)
		return
	}

	ctx.JSON(http.StatusCreated, user)
}

// POST /api/users/login — заглушка для ЛР4
func (h *Handler) LoginUser(ctx *gin.Context) {
	ctx.Status(http.StatusOK)
}

// POST /api/users/logout — заглушка для ЛР4
func (h *Handler) LogoutUser(ctx *gin.Context) {
	ctx.Status(http.StatusOK)
}
