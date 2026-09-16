package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"cepheids-backend/internal/app/ds"
)

// ---------- Чтение ----------

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

// GetNextClass — следующая опубликованная, с заворотом на первую
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

// GetFilteredClasses — фильтрация по диапазонам pl_slope и pl_intercept
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

// ---------- Создание и публикация через ORM ----------

// CreateDraft — создаёт черновик для пользователя
func (r *Repository) CreateDraft(creatorID uint, name string) (ds.SpectralClass, error) {
	c := ds.SpectralClass{
		Status:    ds.StatusDraft,
		Name:      name,
		CreatorID: creatorID,
	}
	err := r.db.Create(&c).Error
	return c, err
}

// PublishClass — смена статуса на published и сохранение полей
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

// ---------- Логическое удаление через SQL-курсор (без ORM) ----------

// DeleteClassSQL — soft-delete через «сырой» SQL и курсор
func (r *Repository) DeleteClassSQL(id int) error {
	query := `
		UPDATE spectral_classes
		SET status = 'deleted', updated_at = NOW()
		WHERE id = $1 AND status != 'deleted'
		RETURNING id`

	row := r.db.Raw(query, id).Row()

	var returnedID int
	if err := row.Scan(&returnedID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("класс %d не найден или уже удалён", id)
		}
		return fmt.Errorf("ошибка удаления: %w", err)
	}
	return nil
}

// ---------- Лайки ----------

// GetLikesCount — количество лайков у класса
func (r *Repository) GetLikesCount(classID int) (int64, error) {
	var count int64
	err := r.db.Model(&ds.Like{}).
		Where("spectral_class_id = ?", classID).
		Count(&count).Error
	return count, err
}
