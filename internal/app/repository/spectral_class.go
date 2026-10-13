package repository

import (
	"cepheids-backend/internal/app/ds"
	"database/sql"
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrCepheidForbidden = errors.New("spectral class belongs to another observer")

func (r *Repository) GetFirstClass() (ds.SpectralClass, error) {
	var c ds.SpectralClass
	err := r.db.Where("status = ?", ds.StatusPublished).Order("id asc").First(&c).Error
	return c, err
}
func (r *Repository) GetClassByID(id int) (ds.SpectralClass, error) {
	var c ds.SpectralClass
	err := r.db.Where("id = ? AND status = ?", id, ds.StatusPublished).First(&c).Error
	return c, err
}
func (r *Repository) GetNextClass(id int) (ds.SpectralClass, error) {
	var next ds.SpectralClass
	err := r.db.Where("status = ? AND id > ?", ds.StatusPublished, id).Order("id asc").First(&next).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r.GetFirstClass()
	}
	return next, err
}
func (r *Repository) GetDraftClass(creatorID uint) (ds.SpectralClass, error) {
	var c ds.SpectralClass
	err := r.db.Where("status = ? AND creator_id = ?", ds.StatusDraft, creatorID).First(&c).Error
	return c, err
}
func (r *Repository) GetFilteredClasses(minSlope, maxSlope, minB, maxB *float64) ([]ds.SpectralClass, error) {
	classes := make([]ds.SpectralClass, 0)
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
func (r *Repository) CreateCepheid(creatorID uint, name, imageKey, videoKey string) (ds.SpectralClass, error) {
	c := ds.SpectralClass{Status: ds.StatusDraft, Name: name, ImageKey: imageKey, VideoKey: videoKey, CreatorID: creatorID}
	err := r.db.Create(&c).Error
	return c, err
}
func (r *Repository) PublishClass(id int, creatorID uint, description string, plSlope, plIntercept float64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var c ds.SpectralClass
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND status != ?", id, ds.StatusDeleted).First(&c).Error; err != nil {
			return err
		}
		if c.CreatorID != creatorID {
			return ErrCepheidForbidden
		}
		if c.Status != ds.StatusDraft {
			return gorm.ErrRecordNotFound
		}
		return tx.Model(&c).Updates(map[string]interface{}{"status": ds.StatusPublished, "description": description, "pl_slope": plSlope, "pl_intercept": plIntercept}).Error
	})
}
func (r *Repository) DeleteClassSQL(id int, creatorID uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Soft delete remains SQL UPDATE through a cursor, as required by Lab 2.
		row := tx.Raw("SELECT creator_id FROM spectral_classes WHERE id = $1 AND status != 'deleted' FOR UPDATE", id).Row()
		var owner uint
		if err := row.Scan(&owner); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return gorm.ErrRecordNotFound
			}
			return err
		}
		if owner != creatorID {
			return ErrCepheidForbidden
		}
		var returnedID int
		return tx.Raw("UPDATE spectral_classes SET status = 'deleted', updated_at = NOW() WHERE id = $1 AND creator_id = $2 AND status != 'deleted' RETURNING id", id, creatorID).Row().Scan(&returnedID)
	})
}
func (r *Repository) GetLikesCount(classID int) (int64, error) {
	var count int64
	err := r.db.Model(&ds.Like{}).Where("spectral_class_id = ?", classID).Count(&count).Error
	return count, err
}
func (r *Repository) IsLikedByUser(classID int, userID uint) (bool, error) {
	var count int64
	err := r.db.Model(&ds.Like{}).Where("spectral_class_id = ? AND user_id = ?", classID, userID).Count(&count).Error
	return count > 0, err
}
func (r *Repository) AddLike(userID uint, classID int) error {
	like := ds.Like{UserID: userID, SpectralClassID: classID}
	return r.db.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}, {Name: "spectral_class_id"}}, DoNothing: true}).Create(&like).Error
}
func (r *Repository) RemoveLike(userID uint, classID int) error {
	return r.db.Where("user_id = ? AND spectral_class_id = ?", userID, classID).Delete(&ds.Like{}).Error
}
