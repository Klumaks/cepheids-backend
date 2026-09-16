package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"cepheids-backend/internal/app/ds"
)

// Порог: сколько рун показывать до кнопки «Больше»
const descPreviewLen = 20

// previewRunes обрезает строку по рунам (кириллица считается как 1 символ)
// Возвращает превью и флаг: true — есть что скрывать
func previewRunes(s string, n int) (string, bool) {
	r := []rune(s)
	if len(r) <= n {
		return s, false
	}
	return string(r[:n]), true
}

// mediaURL — URL из MinIO или дефолтный с SSR-сервера, если ключ пустой
func mediaURL(key, defaultPath string) string {
	if key == "" {
		return defaultPath
	}
	return minioURL + "/" + key
}

// ---------- GET /feed и /feed/:id ----------
func (h *Handler) Feed(ctx *gin.Context) {
	idStr := ctx.Param("id")
	var (
		c   ds.SpectralClass
		err error
	)

	if idStr == "" {
		// первый опубликованный
		var all []ds.SpectralClass
		all, err = h.Repository.GetPublishedClasses()
		if err == nil && len(all) > 0 {
			c = all[0]
		} else {
			err = http.ErrNoLocation
		}
	} else {
		id, _ := strconv.Atoi(idStr)
		if ctx.Query("next") == "true" {
			c, err = h.Repository.GetNextClass(id)
		} else {
			c, err = h.Repository.GetClassByID(id)
		}
	}

	if err != nil {
		logrus.Error(err)
		ctx.Redirect(http.StatusFound, "/feed")
		return
	}

	likes, _ := h.Repository.GetLikesCount(c.ID)

	preview, isLong := previewRunes(c.Description, descPreviewLen)

	ctx.HTML(http.StatusOK, "feed.html", gin.H{
		"Class":       c,
		"VideoURL":    mediaURL(c.VideoKey, "/static/img/default.mp4"),
		"ImageURL":    mediaURL(c.ImageKey, "/static/img/default.jpg"),
		"Likes":       likes,
		"DescPreview": preview,
		"IsLong":      isLong,
	})
}

// ---------- GET /add ----------
func (h *Handler) Add(ctx *gin.Context) {
	draft, err := h.Repository.GetDraftClass(currentUserID)
	if err != nil {
		// черновика нет — форма создания с дефолтными медиа
		ctx.HTML(http.StatusOK, "add.html", gin.H{
			"HasDraft": false,
			"Draft":    ds.SpectralClass{},
			"ImageURL": mediaURL("", "/static/img/default.jpg"),
			"VideoURL": mediaURL("", "/static/img/default.mp4"),
		})
		return
	}

	ctx.HTML(http.StatusOK, "add.html", gin.H{
		"HasDraft": true,
		"Draft":    draft,
		"ImageURL": mediaURL(draft.ImageKey, "/static/img/default.jpg"),
		"VideoURL": mediaURL(draft.VideoKey, "/static/img/default.mp4"),
	})
}

// ---------- GET /classes ----------
type ClassCard struct {
	ID         int
	Name       string
	ImageURL   string
	Slope      float64
	Intercept  float64
	LikesCount int64
}

func (h *Handler) Classes(ctx *gin.Context) {
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
		logrus.Error(err)
	}

	cards := []ClassCard{}
	for _, c := range classes {
		likes, _ := h.Repository.GetLikesCount(c.ID)
		cards = append(cards, ClassCard{
			ID:         c.ID,
			Name:       c.Name,
			ImageURL:   mediaURL(c.ImageKey, "/static/img/default.jpg"),
			Slope:      c.PlSlope,
			Intercept:  c.PlIntercept,
			LikesCount: likes,
		})
	}

	ctx.HTML(http.StatusOK, "classes.html", gin.H{
		"Cards":    cards,
		"MinSlope": minSlopeStr,
		"MaxSlope": maxSlopeStr,
		"MinB":     minBStr,
		"MaxB":     maxBStr,
	})
}

// ---------- POST /add (создание черновика) ----------
func (h *Handler) CreateDraft(ctx *gin.Context) {
	name := ctx.PostForm("name")
	if name == "" {
		ctx.Redirect(http.StatusFound, "/add")
		return
	}
	_, err := h.Repository.CreateDraft(currentUserID, name)
	if err != nil {
		logrus.Error(err)
	}
	ctx.Redirect(http.StatusFound, "/add")
}

// ---------- POST /publish (публикация черновика) ----------
func (h *Handler) Publish(ctx *gin.Context) {
	idStr := ctx.PostForm("class_id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		ctx.Redirect(http.StatusFound, "/add")
		return
	}

	description := ctx.PostForm("description")
	slopeStr := ctx.PostForm("pl_slope")
	interceptStr := ctx.PostForm("pl_intercept")

	slope, err1 := strconv.ParseFloat(slopeStr, 64)
	intercept, err2 := strconv.ParseFloat(interceptStr, 64)
	if err1 != nil || err2 != nil || description == "" {
		ctx.Redirect(http.StatusFound, "/add")
		return
	}

	if err := h.Repository.PublishClass(id, currentUserID, description, slope, intercept); err != nil {
		logrus.Error(err)
		ctx.Redirect(http.StatusFound, "/add")
		return
	}

	ctx.Redirect(http.StatusFound, "/classes")
}

// ---------- POST /classes/delete (soft-delete через SQL-курсор) ----------
func (h *Handler) DeleteClass(ctx *gin.Context) {
	idStr := ctx.PostForm("class_id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}
	if err := h.Repository.DeleteClassSQL(id); err != nil {
		logrus.Error(err)
	}
	ctx.Redirect(http.StatusFound, "/classes")
}
