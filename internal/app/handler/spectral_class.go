package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"cepheids-backend/internal/app/ds"
)

func (h *Handler) mediaURL(key string) string {
	if key == "" {
		return ""
	}
	return minioURL + "/" + key
}

// GET /api/cepheids — список с фильтрацией (только опубликованные)
func (h *Handler) GetCepheids(ctx *gin.Context) {
	minSlopeStr := ctx.Query("min_slope")
	maxSlopeStr := ctx.Query("max_slope")
	minBStr := ctx.Query("min_b")
	maxBStr := ctx.Query("max_b")

	parseFloat := func(s string) *float64 {
		if s == "" {
			return nil
		}
		if v, e := strconv.ParseFloat(s, 64); e == nil {
			return &v
		}
		return nil
	}

	minSlope := parseFloat(minSlopeStr)
	maxSlope := parseFloat(maxSlopeStr)
	minB := parseFloat(minBStr)
	maxB := parseFloat(maxBStr)

	classes, err := h.Repository.GetFilteredClasses(minSlope, maxSlope, minB, maxB)
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}

	currentUser := GetCurrentUserID()
	for i := range classes {
		classes[i].ImageURL = h.mediaURL(classes[i].ImageKey)
		classes[i].VideoURL = h.mediaURL(classes[i].VideoKey)
		likes, _ := h.Repository.GetLikesCount(classes[i].ID)
		classes[i].LikesCount = likes
		classes[i].IsMine = (classes[i].CreatorID == currentUser)
		classes[i].IsLiked, _ = h.Repository.IsLikedByUser(classes[i].ID, currentUser)
	}

	ctx.JSON(http.StatusOK, classes)
}

// GET /api/cepheids/feed — лента (первый опубликованный)
func (h *Handler) GetFeed(ctx *gin.Context) {
	classes, err := h.Repository.GetPublishedClasses()
	if err != nil || len(classes) == 0 {
		ctx.Status(http.StatusNotFound)
		return
	}

	c := classes[0]
	currentUser := GetCurrentUserID()
	c.ImageURL = h.mediaURL(c.ImageKey)
	c.VideoURL = h.mediaURL(c.VideoKey)
	likes, _ := h.Repository.GetLikesCount(c.ID)
	c.LikesCount = likes
	c.IsMine = (c.CreatorID == currentUser)
	c.IsLiked, _ = h.Repository.IsLikedByUser(c.ID, currentUser)

	ctx.JSON(http.StatusOK, c)
}

// GET /api/cepheids/feed/:id — лента по ID + ?next=true
func (h *Handler) GetFeedByID(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	var c ds.SpectralClass
	if ctx.Query("next") == "true" {
		c, err = h.Repository.GetNextClass(id)
	} else {
		c, err = h.Repository.GetClassByID(id)
	}

	if err != nil {
		ctx.Status(http.StatusNotFound)
		return
	}

	currentUser := GetCurrentUserID()
	c.ImageURL = h.mediaURL(c.ImageKey)
	c.VideoURL = h.mediaURL(c.VideoKey)
	likes, _ := h.Repository.GetLikesCount(c.ID)
	c.LikesCount = likes
	c.IsMine = (c.CreatorID == currentUser)
	c.IsLiked, _ = h.Repository.IsLikedByUser(c.ID, currentUser)

	ctx.JSON(http.StatusOK, c)
}

// GET /api/cepheids/draft — получение черновика текущего пользователя
func (h *Handler) GetDraft(ctx *gin.Context) {
	draft, err := h.Repository.GetDraftClass(GetCurrentUserID())
	if err != nil {
		ctx.Status(http.StatusNotFound)
		return
	}

	draft.ImageURL = h.mediaURL(draft.ImageKey)
	draft.VideoURL = h.mediaURL(draft.VideoKey)

	ctx.JSON(http.StatusOK, draft)
}

// POST /api/cepheids — создание + загрузка файлов
func (h *Handler) CreateCepheid(ctx *gin.Context) {
	name := ctx.PostForm("name")
	if name == "" {
		ctx.Status(http.StatusBadRequest)
		return
	}

	// Получаем файлы
	imageFile, _ := ctx.FormFile("image")
	videoFile, _ := ctx.FormFile("video")

	// Генерируем имена файлов
	imageKey := ""
	videoKey := ""

	if imageFile != nil {
		imageKey = fmt.Sprintf("cepheid_%d_%d.jpg", GetCurrentUserID(), time.Now().Unix())
		uploadedKey, err := h.MinioClient.UploadFile(imageFile, imageKey)
		if err != nil {
			logrus.Error(err)
			ctx.Status(http.StatusInternalServerError)
			return
		}
		imageKey = uploadedKey
	}

	if videoFile != nil {
		videoKey = fmt.Sprintf("cepheid_%d_%d.mp4", GetCurrentUserID(), time.Now().Unix())
		uploadedKey, err := h.MinioClient.UploadFile(videoFile, videoKey)
		if err != nil {
			logrus.Error(err)
			ctx.Status(http.StatusInternalServerError)
			return
		}
		videoKey = uploadedKey
	}

	// Создаем в БД
	cepheid, err := h.Repository.CreateCepheid(GetCurrentUserID(), name, imageKey, videoKey)
	if err != nil {
		logrus.Error(err)
		ctx.Status(http.StatusInternalServerError)
		return
	}

	cepheid.ImageURL = h.mediaURL(cepheid.ImageKey)
	cepheid.VideoURL = h.mediaURL(cepheid.VideoKey)

	ctx.JSON(http.StatusCreated, cepheid)
}

// PUT /api/cepheids/:id/publish — публикация + возврат обновлённого объекта
func (h *Handler) PublishCepheid(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	var req struct {
		Description string  `json:"description"`
		PlSlope     float64 `json:"pl_slope"`
		PlIntercept float64 `json:"pl_intercept"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	if err := h.Repository.PublishClass(id, GetCurrentUserID(), req.Description, req.PlSlope, req.PlIntercept); err != nil {
		ctx.Status(http.StatusNotFound)
		return
	}

	// Дочитываем обновлённую запись — теперь она published, GetClassByID её найдёт
	updated, err := h.Repository.GetClassByID(id)
	if err != nil {
		ctx.Status(http.StatusNotFound)
		return
	}

	// Обогащаем теми же служебными полями, что и в GET/POST
	updated.ImageURL = h.mediaURL(updated.ImageKey)
	updated.VideoURL = h.mediaURL(updated.VideoKey)
	likes, _ := h.Repository.GetLikesCount(updated.ID)
	updated.LikesCount = likes
	updated.IsMine = (updated.CreatorID == GetCurrentUserID())
	updated.IsLiked, _ = h.Repository.IsLikedByUser(updated.ID, GetCurrentUserID())

	ctx.JSON(http.StatusOK, updated) // ← тело, как у POST
}

// DELETE /api/cepheids/:id — удаление (только свои)
func (h *Handler) DeleteCepheid(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	if err := h.Repository.DeleteClassSQL(id, GetCurrentUserID()); err != nil {
		logrus.Error(err)
		ctx.Status(http.StatusForbidden)
		return
	}

	ctx.Status(http.StatusOK)
}

// POST /api/cepheids/:id/like — лайк (0 или 1)
func (h *Handler) LikeCepheid(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	var req struct {
		Like int `json:"like"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	if req.Like == 1 {
		h.Repository.AddLike(GetCurrentUserID(), id)
	} else {
		h.Repository.RemoveLike(GetCurrentUserID(), id)
	}

	ctx.Status(http.StatusOK)
}
