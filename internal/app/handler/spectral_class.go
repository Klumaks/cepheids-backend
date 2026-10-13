package handler

import (
	"cepheids-backend/internal/app/ds"
	"cepheids-backend/internal/app/repository"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"math"
	"net/http"
	"path/filepath"
	"strconv"
	"unicode/utf8"
)

type CepheidPublication struct {
	Description string   `json:"description" example:"Калибровка зависимости период–светимость"`
	PlSlope     *float64 `json:"pl_slope" binding:"required" example:"-2.76"`
	PlIntercept *float64 `json:"pl_intercept" binding:"required" example:"-1.4"`
}
type CepheidLike struct {
	Like *int `json:"like" binding:"required" example:"1"`
}

func (h *Handler) enrichCepheid(c *ds.SpectralClass, ctx *gin.Context) error {
	c.ImageURL = h.MinioClient.GetFileURL(c.ImageKey)
	c.VideoURL = h.MinioClient.GetFileURL(c.VideoKey)
	var err error
	c.LikesCount, err = h.Repository.GetLikesCount(c.ID)
	if err != nil {
		return err
	}
	id := GetCurrentUserID(ctx)
	c.IsMine = 0
	c.IsLiked = 0
	if id != 0 && c.CreatorID == id {
		c.IsMine = 1
	}
	if id != 0 {
		var liked bool
		liked, err = h.Repository.IsLikedByUser(c.ID, id)
		if liked {
			c.IsLiked = 1
		}
	}
	return err
}
func cepheidRepositoryError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		ctx.Status(http.StatusNotFound)
	case errors.Is(err, repository.ErrCepheidForbidden):
		ctx.Status(http.StatusForbidden)
	default:
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			ctx.Status(http.StatusConflict)
		} else {
			ctx.Status(http.StatusInternalServerError)
		}
	}
}
func cepheidID(ctx *gin.Context) (int, bool) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil || id <= 0 {
		ctx.Status(http.StatusBadRequest)
		return 0, false
	}
	return id, true
}
func cepheidRange(ctx *gin.Context, name string) (*float64, bool) {
	raw, exists := ctx.GetQuery(name)
	if !exists {
		return nil, true
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		ctx.Status(http.StatusBadRequest)
		return nil, false
	}
	return &value, true
}

// GetCepheids godoc
// @Summary Спектральные классы цефеид с фильтрацией
// @Tags Спектральные классы
// @Produce json
// @Param min_slope query number false "Минимальный наклон зависимости период–светимость"
// @Param max_slope query number false "Максимальный наклон"
// @Param min_b query number false "Минимальный свободный член"
// @Param max_b query number false "Максимальный свободный член"
// @Success 200 {array} ds.SpectralClass
// @Failure 400 "Некорректный диапазон"
// @Failure 500 "Ошибка БД"
// @Router /cepheids [get]
func (h *Handler) GetCepheids(ctx *gin.Context) {
	minSlope, ok := cepheidRange(ctx, "min_slope")
	if !ok {
		return
	}
	maxSlope, ok := cepheidRange(ctx, "max_slope")
	if !ok {
		return
	}
	minB, ok := cepheidRange(ctx, "min_b")
	if !ok {
		return
	}
	maxB, ok := cepheidRange(ctx, "max_b")
	if !ok {
		return
	}
	if (minSlope != nil && maxSlope != nil && *minSlope > *maxSlope) || (minB != nil && maxB != nil && *minB > *maxB) {
		ctx.Status(http.StatusBadRequest)
		return
	}
	classes, err := h.Repository.GetFilteredClasses(minSlope, maxSlope, minB, maxB)
	if err != nil {
		cepheidRepositoryError(ctx, err)
		return
	}
	for i := range classes {
		if err := h.enrichCepheid(&classes[i], ctx); err != nil {
			cepheidRepositoryError(ctx, err)
			return
		}
	}
	ctx.JSON(http.StatusOK, classes)
}

// GetFeed godoc
// @Summary Первый опубликованный спектральный класс
// @Tags Спектральные классы
// @Produce json
// @Success 200 {object} ds.SpectralClass
// @Failure 404 "Лента пуста"
// @Failure 500 "Ошибка БД"
// @Router /cepheids/feed [get]
func (h *Handler) GetFeed(ctx *gin.Context) {
	c, err := h.Repository.GetFirstClass()
	if err != nil {
		cepheidRepositoryError(ctx, err)
		return
	}
	if err := h.enrichCepheid(&c, ctx); err != nil {
		cepheidRepositoryError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, c)
}

// GetFeedByID godoc
// @Summary Спектральный класс по ID или следующий в ленте
// @Tags Спектральные классы
// @Produce json
// @Param id path int true "ID спектрального класса"
// @Param next query boolean false "Следующий опубликованный класс; в конце переход к первому"
// @Success 200 {object} ds.SpectralClass
// @Failure 400 "Некорректный ID"
// @Failure 404 "Класс не найден"
// @Failure 500 "Ошибка БД"
// @Router /cepheids/feed/{id} [get]
func (h *Handler) GetFeedByID(ctx *gin.Context) {
	id, ok := cepheidID(ctx)
	if !ok {
		return
	}
	var c ds.SpectralClass
	var err error
	if ctx.Query("next") == "true" {
		c, err = h.Repository.GetNextClass(id)
	} else {
		c, err = h.Repository.GetClassByID(id)
	}
	if err != nil {
		cepheidRepositoryError(ctx, err)
		return
	}
	if err := h.enrichCepheid(&c, ctx); err != nil {
		cepheidRepositoryError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, c)
}

// GetDraft godoc
// @Summary Черновик спектрального класса текущего наблюдателя
// @Tags Спектральные классы
// @Security CepheidBearer
// @Produce json
// @Success 200 {object} ds.SpectralClass
// @Failure 401 "Требуется действующий JWT"
// @Failure 404 "Черновика нет"
// @Failure 500 "Ошибка БД"
// @Failure 503 "Redis недоступен"
// @Router /cepheids/draft [get]
func (h *Handler) GetDraft(ctx *gin.Context) {
	c, err := h.Repository.GetDraftClass(GetCurrentUserID(ctx))
	if err != nil {
		cepheidRepositoryError(ctx, err)
		return
	}
	if err := h.enrichCepheid(&c, ctx); err != nil {
		cepheidRepositoryError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, c)
}

// CreateCepheid godoc
// @Summary Создание черновика спектрального класса с файлами
// @Tags Спектральные классы
// @Security CepheidBearer
// @Accept multipart/form-data
// @Produce json
// @Param name formData string true "Название спектрального класса"
// @Param image formData file false "Изображение"
// @Param video formData file false "Короткое видео"
// @Success 201 {object} ds.SpectralClass
// @Failure 400 "Некорректные данные"
// @Failure 401 "Требуется действующий JWT"
// @Failure 409 "Черновик уже существует"
// @Failure 500 "Ошибка БД или MinIO"
// @Failure 503 "Redis недоступен"
// @Router /cepheids [post]
func (h *Handler) CreateCepheid(ctx *gin.Context) {
	name := ctx.PostForm("name")
	if name == "" || utf8.RuneCountInString(name) > 100 {
		ctx.Status(http.StatusBadRequest)
		return
	}
	id := GetCurrentUserID(ctx)
	if _, err := h.Repository.GetDraftClass(id); err == nil {
		ctx.Status(http.StatusConflict)
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		cepheidRepositoryError(ctx, err)
		return
	}
	imageKey, videoKey := "", ""
	for _, media := range []string{"image", "video"} {
		file, err := ctx.FormFile(media)
		if errors.Is(err, http.ErrMissingFile) {
			continue
		}
		if err != nil {
			ctx.Status(http.StatusBadRequest)
			return
		}
		extension := filepath.Ext(file.Filename)
		if media == "image" && extension != ".jpg" && extension != ".jpeg" && extension != ".png" && extension != ".webp" {
			extension = ".jpg"
		}
		if media == "video" && extension != ".mp4" && extension != ".webm" {
			extension = ".mp4"
		}
		key := fmt.Sprintf("cepheid_%d_%s%s", id, uuid.NewString(), extension)
		uploaded, err := h.MinioClient.UploadFile(file, key)
		if err != nil {
			ctx.Status(http.StatusInternalServerError)
			return
		}
		if media == "image" {
			imageKey = uploaded
		} else {
			videoKey = uploaded
		}
	}
	c, err := h.Repository.CreateCepheid(id, name, imageKey, videoKey)
	if err != nil {
		cepheidRepositoryError(ctx, err)
		return
	}
	if err := h.enrichCepheid(&c, ctx); err != nil {
		cepheidRepositoryError(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, c)
}

// PublishCepheid godoc
// @Summary Публикация своего черновика спектрального класса
// @Tags Спектральные классы
// @Security CepheidBearer
// @Accept json
// @Produce json
// @Param id path int true "ID спектрального класса"
// @Param calibration body CepheidPublication true "Описание и коэффициенты период–светимость"
// @Success 200 {object} ds.SpectralClass
// @Failure 400 "Некорректные данные"
// @Failure 401 "Требуется действующий JWT"
// @Failure 403 "Чужой спектральный класс"
// @Failure 404 "Черновик не найден"
// @Failure 500 "Ошибка БД"
// @Failure 503 "Redis недоступен"
// @Router /cepheids/{id}/publish [put]
func (h *Handler) PublishCepheid(ctx *gin.Context) {
	id, ok := cepheidID(ctx)
	if !ok {
		return
	}
	var req CepheidPublication
	if ctx.ShouldBindJSON(&req) != nil || req.PlSlope == nil || req.PlIntercept == nil || utf8.RuneCountInString(req.Description) > 500 {
		ctx.Status(http.StatusBadRequest)
		return
	}
	if err := h.Repository.PublishClass(id, GetCurrentUserID(ctx), req.Description, *req.PlSlope, *req.PlIntercept); err != nil {
		cepheidRepositoryError(ctx, err)
		return
	}
	c, err := h.Repository.GetClassByID(id)
	if err != nil {
		cepheidRepositoryError(ctx, err)
		return
	}
	if err := h.enrichCepheid(&c, ctx); err != nil {
		cepheidRepositoryError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, c)
}

// DeleteCepheid godoc
// @Summary Логическое удаление своего спектрального класса
// @Tags Спектральные классы
// @Security CepheidBearer
// @Param id path int true "ID спектрального класса"
// @Success 200 "Спектральный класс удалён"
// @Failure 400 "Некорректный ID"
// @Failure 401 "Требуется действующий JWT"
// @Failure 403 "Чужой спектральный класс"
// @Failure 404 "Класс не найден"
// @Failure 500 "Ошибка БД"
// @Failure 503 "Redis недоступен"
// @Router /cepheids/{id} [delete]
func (h *Handler) DeleteCepheid(ctx *gin.Context) {
	id, ok := cepheidID(ctx)
	if !ok {
		return
	}
	if err := h.Repository.DeleteClassSQL(id, GetCurrentUserID(ctx)); err != nil {
		cepheidRepositoryError(ctx, err)
		return
	}
	ctx.Status(http.StatusOK)
}

// LikeCepheid godoc
// @Summary Поставить или отменить лайк спектрального класса
// @Tags Спектральные классы
// @Security CepheidBearer
// @Accept json
// @Param id path int true "ID опубликованного спектрального класса"
// @Param appreciation body CepheidLike true "1 — поставить, 0 — отменить"
// @Success 200 "Лайк обновлён"
// @Failure 400 "Допускаются только 0 и 1"
// @Failure 401 "Требуется действующий JWT"
// @Failure 404 "Класс не найден"
// @Failure 500 "Ошибка БД"
// @Failure 503 "Redis недоступен"
// @Router /cepheids/{id}/like [post]
func (h *Handler) LikeCepheid(ctx *gin.Context) {
	id, ok := cepheidID(ctx)
	if !ok {
		return
	}
	var req CepheidLike
	if ctx.ShouldBindJSON(&req) != nil || req.Like == nil || (*req.Like != 0 && *req.Like != 1) {
		ctx.Status(http.StatusBadRequest)
		return
	}
	if _, err := h.Repository.GetClassByID(id); err != nil {
		cepheidRepositoryError(ctx, err)
		return
	}
	var err error
	if *req.Like == 1 {
		err = h.Repository.AddLike(GetCurrentUserID(ctx), id)
	} else {
		err = h.Repository.RemoveLike(GetCurrentUserID(ctx), id)
	}
	if err != nil {
		cepheidRepositoryError(ctx, err)
		return
	}
	ctx.Status(http.StatusOK)
}
