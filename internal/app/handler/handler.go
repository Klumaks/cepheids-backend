package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"cepheids-backend/internal/app/repository"
)

const minioURL = "http://localhost:9000/cepheids"

type Handler struct{ Repository *repository.Repository }

func NewHandler(r *repository.Repository) *Handler { return &Handler{Repository: r} }

// ClassCard — карточка плитки
type ClassCard struct {
	ID         int
	Name       string
	ImageURL   string
	Slope      float64
	Intercept  float64 // свободный член b
	LikesCount int
}

func (h *Handler) Feed(ctx *gin.Context) {
	idStr := ctx.Param("id")
	var (
		c   repository.SpectralClass
		err error
	)
	if idStr == "" { // панель вкладок без ID — первая опубликованная
		var all []repository.SpectralClass
		all, err = h.Repository.GetPublishedClasses()
		if err == nil && len(all) > 0 {
			c = all[0]
		} else {
			err = fmt.Errorf("нет опубликованных классов")
		}
	} else {
		id, _ := strconv.Atoi(idStr)
		if ctx.Query("next") == "true" {
			c, err = h.Repository.GetNextClass(id)
		} else {
			c, err = h.Repository.GetClassByID(id)
		}
	}
	if err != nil { // удалённые и несуществующие не смотрим
		logrus.Error(err)
		ctx.Redirect(http.StatusFound, "/feed")
		return
	}
	ctx.HTML(http.StatusOK, "feed.html", gin.H{
		"Class":    c,
		"VideoURL": minioURL + "/" + c.VideoKey,
		"ImageURL": minioURL + "/" + c.ImageKey,
		"Likes":    len(c.Likes), // количество лайков считаем в контроллере
	})
}

func (h *Handler) Add(ctx *gin.Context) {
	d, err := h.Repository.GetDraftClass()
	if err != nil {
		logrus.Error(err)
		ctx.Redirect(http.StatusFound, "/feed")
		return
	}
	ctx.HTML(http.StatusOK, "add.html", gin.H{
		"Draft":    d,
		"ImageURL": minioURL + "/" + d.ImageKey,
		"VideoURL": minioURL + "/" + d.VideoKey,
	})
}

func (h *Handler) Classes(ctx *gin.Context) {
	minSlopeStr := ctx.Query("min_slope")
	maxSlopeStr := ctx.Query("max_slope")
	minBStr := ctx.Query("min_b")
	maxBStr := ctx.Query("max_b")

	var minSlope, maxSlope, minB, maxB *float64
	if minSlopeStr != "" {
		if v, e := strconv.ParseFloat(minSlopeStr, 64); e == nil {
			minSlope = &v
		}
	}
	if maxSlopeStr != "" {
		if v, e := strconv.ParseFloat(maxSlopeStr, 64); e == nil {
			maxSlope = &v
		}
	}
	if minBStr != "" {
		if v, e := strconv.ParseFloat(minBStr, 64); e == nil {
			minB = &v
		}
	}
	if maxBStr != "" {
		if v, e := strconv.ParseFloat(maxBStr, 64); e == nil {
			maxB = &v
		}
	}

	classes, err := h.Repository.GetFiltered(minSlope, maxSlope, minB, maxB)
	if err != nil {
		logrus.Error(err)
	}

	cards := []ClassCard{}
	for _, c := range classes {
		cards = append(cards, ClassCard{
			ID:         c.ID,
			Name:       c.Name,
			ImageURL:   minioURL + "/" + c.ImageKey,
			Slope:      c.PlSlope,
			Intercept:  c.PlIntercept,
			LikesCount: len(c.Likes),
		})
	}

	ctx.HTML(http.StatusOK, "classes.html", gin.H{
		"Cards":    cards,
		"MinSlope": minSlopeStr, // поля поиска сохраняются после запроса
		"MaxSlope": maxSlopeStr,
		"MinB":     minBStr,
		"MaxB":     maxBStr,
	})
}
