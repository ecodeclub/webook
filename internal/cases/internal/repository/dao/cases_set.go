package dao

import (
	"context"
	"time"

	"github.com/ego-component/egorm"
	"gorm.io/gorm"
)

type CaseSetDAO interface {
	Create(ctx context.Context, cs CaseSet) (int64, error)
	GetByID(ctx context.Context, id int64) (CaseSet, error)

	GetCasesByID(ctx context.Context, id int64) ([]Case, error)
	UpdateCasesByID(ctx context.Context, id int64, cids []int64) error

	Count(ctx context.Context) (int64, error)
	CountByBiz(ctx context.Context, biz string) (int64, error)
	List(ctx context.Context, offset, limit int) ([]CaseSet, error)
	UpdateNonZero(ctx context.Context, set CaseSet) error
	GetByIDs(ctx context.Context, ids []int64) ([]CaseSet, error)

	ListByBiz(ctx context.Context, offset int, limit int, biz string) ([]CaseSet, error)
	GetByBiz(ctx context.Context, biz string, bizId int64) (CaseSet, error)
	GetRefCasesByIDs(ctx context.Context, ids []int64) ([]CaseSetCase, error)
}

type caseSetDAO struct {
	db *egorm.Component
}

func (c *caseSetDAO) CountByBiz(ctx context.Context, biz string) (int64, error) {
	var count int64
	db := c.db.WithContext(ctx)
	err := db.
		Model(&CaseSet{}).
		Where("biz = ?", biz).Count(&count).Error
	return count, err
}

func (c *caseSetDAO) GetRefCasesByIDs(ctx context.Context, ids []int64) ([]CaseSetCase, error) {
	var res []CaseSetCase
	err := c.db.WithContext(ctx).Where("cs_id IN ?", ids).Find(&res).Error
	return res, err
}

func (c *caseSetDAO) ListByBiz(ctx context.Context, offset int, limit int, biz string) ([]CaseSet, error) {
	var res []CaseSet
	db := c.db.WithContext(ctx)
	err := db.Where("biz = ?", biz).
		Offset(offset).Limit(limit).Order("id DESC").Find(&res).Error
	return res, err
}

func (c *caseSetDAO) GetByBiz(ctx context.Context, biz string, bizId int64) (CaseSet, error) {
	var res CaseSet
	db := c.db.WithContext(ctx)
	err := db.Where("biz = ? AND biz_id = ?", biz, bizId).
		Order("utime DESC").
		First(&res).Error
	return res, err
}

func (c *caseSetDAO) Create(ctx context.Context, cs CaseSet) (int64, error) {
	now := time.Now().UnixMilli()
	cs.Ctime = now
	cs.Utime = now
	err := c.db.WithContext(ctx).Create(&cs).Error
	return cs.Id, err
}

func (c *caseSetDAO) GetByID(ctx context.Context, id int64) (CaseSet, error) {
	var cs CaseSet
	err := c.db.WithContext(ctx).Where("id = ?", id).First(&cs).Error
	return cs, err
}

func (c *caseSetDAO) GetCasesByID(ctx context.Context, id int64) ([]Case, error) {
	var cids []int64
	err := c.db.WithContext(ctx).
		Select("cid").
		Model(&CaseSetCase{}).
		Where("cs_id = ?", id).
		Scan(&cids).Error
	if err != nil {
		return nil, err
	}
	var cs []Case
	err = c.db.WithContext(ctx).
		Model(&Case{}).
		Where("id in ?", cids).
		Scan(&cs).Error
	return cs, err

}

func (c *caseSetDAO) UpdateCasesByID(ctx context.Context, id int64, cids []int64) error {
	return c.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var cs CaseSet
		if err := tx.WithContext(ctx).First(&cs, "id = ? ", id).Error; err != nil {
			return err
		}
		// 全部删除
		if err := tx.WithContext(ctx).Where("cs_id = ?", id).Delete(&CaseSetCase{}).Error; err != nil {
			return err
		}

		if len(cids) == 0 {
			return nil
		}

		// 重新创建
		now := time.Now().UnixMilli()
		var newQuestions []CaseSetCase
		for i := range cids {
			newQuestions = append(newQuestions, CaseSetCase{
				CSID:  id,
				CID:   cids[i],
				Ctime: now,
				Utime: now,
			})
		}
		return tx.WithContext(ctx).Create(&newQuestions).Error
	})
}

func (c *caseSetDAO) Count(ctx context.Context) (int64, error) {
	var res int64
	db := c.db.WithContext(ctx).Model(&CaseSet{})
	err := db.Select("COUNT(id)").Count(&res).Error
	return res, err
}

func (c *caseSetDAO) List(ctx context.Context, offset, limit int) ([]CaseSet, error) {
	var res []CaseSet
	db := c.db.WithContext(ctx)
	err := db.Offset(offset).Limit(limit).Order("id DESC").Find(&res).Error
	return res, err
}

func (c *caseSetDAO) UpdateNonZero(ctx context.Context, set CaseSet) error {
	set.Utime = time.Now().UnixMilli()
	return c.db.WithContext(ctx).Where("id = ?", set.Id).Updates(set).Error
}

func (c *caseSetDAO) GetByIDs(ctx context.Context, ids []int64) ([]CaseSet, error) {
	var res []CaseSet
	err := c.db.Model(&CaseSet{}).WithContext(ctx).Where("id in (?)", ids).Find(&res).Error
	return res, err
}

func NewCaseSetDAO(db *egorm.Component) CaseSetDAO {
	return &caseSetDAO{db: db}
}
