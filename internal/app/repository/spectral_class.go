package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"cepheids-backend/internal/app/ds"
)

// GetPublishedClasses — все опубликованные
func (r *Repository) GetPublishedClasses() ([]ds.SpectralClass, error) {
	var classes []ds.SpectralClass
	err := r.db.
		Where("status = ?", ds.StatusPublished).
		Order("id asc").
		Find(&classes).Error
	return classes, err
}

// GetClassByID — услуга по ID, только опубликованные
func (r *Repository) GetClassByID(id int) (ds.SpectralClass, error) {
	var c ds.SpectralClass
	err := r.db.
		Where("id = ? AND status = ?", id, ds.StatusPublished).
		First(&c).Error
	return c, err
}

// GetNextClass — следующая опубликованная
func (r *Repository) GetNextClass(id int) (ds.SpectralClass, error) {
	var next ds.SpectralClass
	err := r.db.
		Where("status = ? AND id > ?", ds.StatusPublished, id).
		Order("id asc").
		First(&next).Error
	if err == nil {
		return next, nil
	}
	err = r.db.
		Where("status = ?", ds.StatusPublished).
		Order("id asc").
		First(&next).Error
	return next, err
}

// GetDraftClass — черновик конкретного пользователя
func (r *Repository) GetDraftClass(creatorID uint) (ds.SpectralClass, error) {
	var c ds.SpectralClass
	err := r.db.
		Where("status = ? AND creator_id = ?", ds.StatusDraft, creatorID).
		First(&c).Error
	return c, err
}

// GetFilteredClasses — фильтрация по диапазонам
func (r *Repository) GetFilteredClasses(minSlope, maxSlope, minB, maxB *float64) ([]ds.SpectralClass, error) {
	var classes []ds.SpectralClass
	query := r.db.Where("status = ?", ds.StatusPublished)

	if minSlope != nil {
		query = query.Where("pl_slope >= ?", *minSlope)
	}
	if maxSlope != nil {
		query = query.Where("pl_slope <= ?", *maxSlope)
	}
	if minB != nil {
		query = query.Where("pl_intercept >= ?", *minB)
	}
	if maxB != nil {
		query = query.Where("pl_intercept <= ?", *maxB)
	}

	err := query.Order("id asc").Find(&classes).Error
	return classes, err
}

// CreateCepheid — создание черновика с файлами
func (r *Repository) CreateCepheid(creatorID uint, name, imageKey, videoKey string) (ds.SpectralClass, error) {
	c := ds.SpectralClass{
		Status:    ds.StatusDraft,
		Name:      name,
		ImageKey:  imageKey,
		VideoKey:  videoKey,
		CreatorID: creatorID,
	}
	err := r.db.Create(&c).Error
	return c, err
}

// CreateDraft — создание черновика (старый метод для совместимости)
func (r *Repository) CreateDraft(creatorID uint, name string) (ds.SpectralClass, error) {
	return r.CreateCepheid(creatorID, name, "", "")
}

// PublishClass — публикация
func (r *Repository) PublishClass(id int, creatorID uint, description string, plSlope, plIntercept float64) error {
	res := r.db.Model(&ds.SpectralClass{}).
		Where("id = ? AND creator_id = ? AND status = ?", id, creatorID, ds.StatusDraft).
		Updates(map[string]interface{}{
			"status":       ds.StatusPublished,
			"description":  description,
			"pl_slope":     plSlope,
			"pl_intercept": plIntercept,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("черновик %d не найден или уже опубликован", id)
	}
	return nil
}

// DeleteClassSQL — удаление (только свои)
func (r *Repository) DeleteClassSQL(id int, creatorID uint) error {
	query := `
		UPDATE spectral_classes
		SET status = 'deleted', updated_at = NOW()
		WHERE id = $1 AND creator_id = $2 AND status != 'deleted'
		RETURNING id`

	row := r.db.Raw(query, id, creatorID).Row()

	var returnedID int
	if err := row.Scan(&returnedID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("класс %d не найден или не принадлежит пользователю", id)
		}
		return fmt.Errorf("ошибка удаления: %w", err)
	}
	return nil
}

// GetLikesCount — количество лайков
func (r *Repository) GetLikesCount(classID int) (int64, error) {
	var count int64
	err := r.db.Model(&ds.Like{}).
		Where("spectral_class_id = ?", classID).
		Count(&count).Error
	return count, err
}

// IsLikedByUser — лайкнул ли пользователь
func (r *Repository) IsLikedByUser(classID int, userID uint) (bool, error) {
	var count int64
	err := r.db.Model(&ds.Like{}).
		Where("spectral_class_id = ? AND user_id = ?", classID, userID).
		Count(&count).Error
	return count > 0, err
}

// AddLike — добавить лайк
func (r *Repository) AddLike(userID uint, classID int) error {
	like := ds.Like{
		UserID:          userID,
		SpectralClassID: classID,
	}
	return r.db.Create(&like).Error
}

// RemoveLike — удалить лайк
func (r *Repository) RemoveLike(userID uint, classID int) error {
	return r.db.Where("user_id = ? AND spectral_class_id = ?", userID, classID).
		Delete(&ds.Like{}).Error
}
