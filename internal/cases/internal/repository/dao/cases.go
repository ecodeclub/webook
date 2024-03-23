package dao

import (
	"context"
	"time"

	"github.com/ego-component/egorm"
	"gorm.io/gorm"
)

type CaseDAO interface {
	// 管理端操作 case表
	Create(ctx context.Context, c Case) (int64, error)
	Update(ctx context.Context, c Case) error
	GetCaseByID(ctx context.Context, id int64) (Case, error)
	List(ctx context.Context, offset, limit int) ([]Case, error)
	Count(ctx context.Context) (int64, error)

	Sync(ctx context.Context, c Case) (int64, error)

	// 线上库
	PublishCaseList(ctx context.Context, offset, limit int) ([]PublishCase, error)
	PublishCaseCount(ctx context.Context) (int64, error)
	GetPublishCase(ctx context.Context, caseId int64) (PublishCase, error)
}

type caseDAO struct {
	db *egorm.Component
}

func NewCaseDao(db *egorm.Component) CaseDAO {
	return &caseDAO{
		db: db,
	}
}
func (ca *caseDAO) Count(ctx context.Context) (int64, error) {
	var res int64
	err := ca.db.WithContext(ctx).Model(&Case{}).Select("COUNT(id)").Count(&res).Error
	return res, err
}

func (ca *caseDAO) Create(ctx context.Context, c Case) (int64, error) {
	c.Ctime = time.Now().UnixMilli()
	c.Utime = time.Now().UnixMilli()
	err := ca.db.WithContext(ctx).Create(&c).Error
	return c.Id, err
}

func (ca *caseDAO) Update(ctx context.Context, c Case) error {
	now := time.Now().UnixMilli()
	return ca.db.WithContext(ctx).
		Model(&Case{}).Where("id = ?", c.Id).Updates(map[string]any{
		"title":     c.Title,
		"content":   c.Content,
		"code_repo": c.CodeRepo,
		"keywords":  c.Keywords,
		"labels":    c.Labels,
		"shorthand": c.Shorthand,
		"highlight": c.Highlight,
		"guidance":  c.Guidance,
		"utime":     now,
	}).Error
}

func (ca *caseDAO) update(ctx context.Context, tx *gorm.DB, c Case) error {
	now := time.Now().UnixMilli()
	return tx.WithContext(ctx).
		Model(&Case{}).Where("id = ?", c.Id).Updates(map[string]any{
		"title":     c.Title,
		"content":   c.Content,
		"code_repo": c.CodeRepo,
		"keywords":  c.Keywords,
		"shorthand": c.Shorthand,
		"highlight": c.Highlight,
		"labels":    c.Labels,
		"guidance":  c.Guidance,
		"utime":     now,
	}).Error

}

func (ca *caseDAO) GetCaseByID(ctx context.Context, id int64) (Case, error) {
	var c Case
	err := ca.db.WithContext(ctx).Where("id = ?", id).First(&c).Error
	return c, err
}

func (ca *caseDAO) List(ctx context.Context, offset, limit int) ([]Case, error) {
	var caseList []Case
	err := ca.db.WithContext(ctx).
		Select("id", "title", "content", "utime").
		Order("id desc").
		Offset(offset).
		Limit(limit).
		Find(&caseList).Error
	return caseList, err
}

func (ca *caseDAO) Sync(ctx context.Context, c Case) (int64, error) {
	id := c.Id
	c.Utime = time.Now().UnixMilli()
	err := ca.db.Transaction(func(tx *gorm.DB) error {
		if c.Id == 0 {
			c.Ctime = time.Now().UnixMilli()
			err := tx.WithContext(ctx).Create(&c).Error
			if err != nil {
				return err
			}
			id = c.Id
		} else {
			err := ca.update(ctx, tx, c)
			if err != nil {
				return err
			}
		}
		return tx.Save(PublishCase(c)).Error
	})
	return id, err
}

func (ca *caseDAO) PublishCaseList(ctx context.Context, offset, limit int) ([]PublishCase, error) {
	publishCaseList := make([]PublishCase, 0, limit)
	err := ca.db.WithContext(ctx).
		Order("id desc").
		Select("id", "title", "content", "utime").
		Offset(offset).
		Limit(limit).
		Find(&publishCaseList).Error
	return publishCaseList, err
}

func (ca *caseDAO) PublishCaseCount(ctx context.Context) (int64, error) {
	var res int64
	err := ca.db.WithContext(ctx).Model(&PublishCase{}).Select("COUNT(id)").Count(&res).Error
	return res, err
}

func (ca *caseDAO) GetPublishCase(ctx context.Context, caseId int64) (PublishCase, error) {
	var c PublishCase
	db := ca.db.WithContext(ctx)
	err := db.Where("id = ?", caseId).First(&c).Error
	return c, err
}
