package handler

import (
	"cepheids-backend/internal/app/ds"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"net/http"
	"strings"
	"unicode/utf8"
)

type CepheidCredentials struct {
	Login    string `json:"login" binding:"required" example:"cepheid_observer"`
	Password string `json:"password" binding:"required" example:"observation2026"`
}
type CepheidLoginResponse struct {
	AccessToken string  `json:"access_token"`
	TokenType   string  `json:"token_type" example:"Bearer"`
	ExpiresIn   int64   `json:"expires_in" example:"3600"`
	User        ds.User `json:"user"`
}

func bindCepheidCredentials(ctx *gin.Context) (CepheidCredentials, bool) {
	var req CepheidCredentials
	if ctx.ShouldBindJSON(&req) != nil {
		ctx.Status(http.StatusBadRequest)
		return req, false
	}
	req.Login = strings.TrimSpace(req.Login)
	if req.Login == "" || utf8.RuneCountInString(req.Login) > 25 || len(req.Password) == 0 || len(req.Password) > 72 {
		ctx.Status(http.StatusBadRequest)
		return req, false
	}
	return req, true
}

// RegisterUser godoc
// @Summary Регистрация наблюдателя цефеид
// @Tags Пользователи
// @Accept json
// @Produce json
// @Param credentials body CepheidCredentials true "Логин и пароль"
// @Success 201 {object} ds.User
// @Failure 400 "Некорректные данные"
// @Failure 409 "Логин занят"
// @Failure 500 "Ошибка БД"
// @Router /users/register [post]
func (h *Handler) RegisterUser(ctx *gin.Context) {
	req, ok := bindCepheidCredentials(ctx)
	if !ok {
		return
	}
	user, err := h.Repository.CreateUser(req.Login, req.Password)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			ctx.Status(http.StatusConflict)
		} else {
			ctx.Status(http.StatusInternalServerError)
		}
		return
	}
	ctx.JSON(http.StatusCreated, user)
}

// LoginUser godoc
// @Summary Вход наблюдателя цефеид
// @Tags Пользователи
// @Accept json
// @Produce json
// @Param credentials body CepheidCredentials true "Логин и пароль"
// @Success 200 {object} CepheidLoginResponse
// @Failure 400 "Некорректные данные"
// @Failure 401 "Неверный логин или пароль"
// @Failure 500 "Ошибка сервера"
// @Router /users/login [post]
func (h *Handler) LoginUser(ctx *gin.Context) {
	req, ok := bindCepheidCredentials(ctx)
	if !ok {
		return
	}
	user, err := h.Repository.GetUserByLogin(req.Login)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		ctx.Status(http.StatusInternalServerError)
		return
	}
	if err != nil || bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)) != nil {
		ctx.Status(http.StatusUnauthorized)
		return
	}
	encoded, err := h.CepheidTokens.Issue(user.ID)
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}
	ctx.JSON(http.StatusOK, CepheidLoginResponse{encoded, "Bearer", int64(h.CepheidTokens.Lifetime.Seconds()), user})
}

// LogoutUser godoc
// @Summary Выход наблюдателя и отзыв JWT
// @Tags Пользователи
// @Security CepheidBearer
// @Success 200 "JWT отозван"
// @Failure 401 "Требуется действующий JWT"
// @Failure 503 "Redis недоступен"
// @Router /users/logout [post]
func (h *Handler) LogoutUser(ctx *gin.Context) {
	value, _ := ctx.Get(cepheidClaimsKey)
	claims, ok := value.(*jwt.RegisteredClaims)
	if !ok {
		ctx.Status(http.StatusUnauthorized)
		return
	}
	if err := h.CepheidTokens.Revoke(ctx.Request.Context(), claims); err != nil {
		ctx.Status(http.StatusServiceUnavailable)
		return
	}
	ctx.Status(http.StatusOK)
}
