package dao

import (
	"context"
	"errors"
	"time"

	"github.com/ego-component/egorm"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ExperienceDAO interface {
	Upsert(ctx context.Context, experience Experience) (int64, error)
	Delete(ctx context.Context, uid int64, id int64) error
	Find(ctx context.Context, uid int64) ([]Experience, error)
}

type experienceDAO struct {
	db *egorm.Component
}

func NewExperienceDAO(db *egorm.Component) ExperienceDAO {
	return &experienceDAO{
		db: db,
	}
}

func (e *experienceDAO) Upsert(ctx context.Context, experience Experience) (int64, error) {
	now := time.Now().UnixMilli()
	experience.Utime = now
	experience.Ctime = now
	err := e.db.WithContext(ctx).Model(&Experience{}).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "uid"}},
			DoUpdates: clause.AssignmentColumns([]string{"start_time", "end_time", "title", "company_name", "location", "responsibilities", "accomplishments", "skills", "utime"}),
		}).Create(&experience).Error
	return experience.ID, err
}

func (e *experienceDAO) Delete(ctx context.Context, uid int64, id int64) error {

	return e.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.WithContext(ctx).Model(&Experience{}).Where("id = ? and uid = ?", id, uid).Delete(&Experience{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected <= 0 {
			return errors.New("删除失败")
		}
		return nil
	})
}

func (e *experienceDAO) Find(ctx context.Context, uid int64) ([]Experience, error) {
	var experiences []Experience
	err := e.db.WithContext(ctx).Where("uid = ?", uid).Order("StartTime desc").Find(&experiences).Error
	return experiences, err
}
