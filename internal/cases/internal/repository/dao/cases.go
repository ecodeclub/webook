package dao

import (
	"context"
	"time"

	"github.com/ecodeclub/webook/internal/cases/internal/domain"

	"gorm.io/gorm/clause"

	"github.com/ego-component/egorm"
	"gorm.io/gorm"
)

type CaseDAO interface {
	// 管理端操作 case表
	Save(ctx context.Context, c Case) (int64, error)
	GetCaseByID(ctx context.Context, id int64) (Case, error)
	List(ctx context.Context, offset, limit int) ([]Case, error)
	Count(ctx context.Context) (int64, error)

	Sync(ctx context.Context, c Case) (Case, error)
	// 提供给同步到知识库用
	Ids(ctx context.Context) ([]int64, error)
	// 线上库
	PublishCaseList(ctx context.Context, offset, limit int) ([]PublishCase, error)
	PublishCaseCount(ctx context.Context, biz string) (int64, error)
	GetPublishCase(ctx context.Context, caseId int64) (PublishCase, error)
	GetPubByIDs(ctx context.Context, ids []int64) ([]PublishCase, error)

	NotInTotal(ctx context.Context, ids []int64) (int64, error)
	NotIn(ctx context.Context, ids []int64, offset int, limit int) ([]Case, error)
}

type caseDAO struct {
	db            *egorm.Component
	listColumns   []string
	updateColumns []string
}

func (ca *caseDAO) Ids(ctx context.Context) ([]int64, error) {
	var ids []int64
	err := ca.db.WithContext(ctx).
		Select("id").
		Model(&Case{}).
		Where("status = ?", domain.PublishedStatus).
		Scan(&ids).Error
	return ids, err
}

func (ca *caseDAO) NotInTotal(ctx context.Context, ids []int64) (int64, error) {
	var res int64
	err := ca.db.WithContext(ctx).
		Model(&Case{}).
		Where("id NOT IN ?", ids).Count(&res).Error
	return res, err
}

func (ca *caseDAO) NotIn(ctx context.Context, ids []int64, offset int, limit int) ([]Case, error) {
	var res []Case
	err := ca.db.WithContext(ctx).
		Model(&Case{}).
		Where("id NOT IN ?", ids).Order("utime DESC").
		Offset(offset).Limit(limit).Find(&res).Error
	return res, err
}

func (ca *caseDAO) Count(ctx context.Context) (int64, error) {
	var res int64
	err := ca.db.WithContext(ctx).Model(&Case{}).Select("COUNT(id)").Count(&res).Error
	return res, err
}

func (ca *caseDAO) Save(ctx context.Context, c Case) (int64, error) {
	return ca.save(ca.db.WithContext(ctx), &c)
}

func (ca *caseDAO) save(db *gorm.DB, c *Case) (int64, error) {
	now := time.Now().UnixMilli()
	c.Utime = now
	c.Ctime = now
	err := db.Clauses(clause.OnConflict{
		DoUpdates: clause.AssignmentColumns(ca.updateColumns),
	}).Create(c).Error
	return c.Id, err
}

func (ca *caseDAO) GetCaseByID(ctx context.Context, id int64) (Case, error) {
	var c Case
	err := ca.db.WithContext(ctx).Where("id = ?", id).First(&c).Error
	return c, err
}

func (ca *caseDAO) List(ctx context.Context, offset, limit int) ([]Case, error) {
	var caseList []Case
	err := ca.db.WithContext(ctx).
		Select(ca.listColumns).
		Order("id desc").
		Offset(offset).
		Limit(limit).
		Find(&caseList).Error
	return caseList, err
}

func (ca *caseDAO) Sync(ctx context.Context, c Case) (Case, error) {
	err := ca.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		id, err := ca.save(tx, &c)
		if err != nil {
			return err
		}
		c.Id = id
		pubC := PublishCase(c)
		return tx.Clauses(clause.OnConflict{
			DoUpdates: clause.AssignmentColumns(ca.updateColumns),
		}).Create(&pubC).Error
	})
	return c, err
}

func (ca *caseDAO) PublishCaseList(ctx context.Context, offset, limit int) ([]PublishCase, error) {
	publishCaseList := make([]PublishCase, 0, limit)
	err := ca.db.WithContext(ctx).
		Order("id desc").
		Select(ca.listColumns).
		Offset(offset).
		Limit(limit).
		Find(&publishCaseList).Error
	return publishCaseList, err
}

func (ca *caseDAO) PublishCaseCount(ctx context.Context, biz string) (int64, error) {
	var res int64
	err := ca.db.WithContext(ctx).Model(&PublishCase{}).Select("COUNT(id)").
		Where("biz = ?", biz).
		Count(&res).Error
	return res, err
}

func (ca *caseDAO) GetPublishCase(ctx context.Context, caseId int64) (PublishCase, error) {
	var c PublishCase
	db := ca.db.WithContext(ctx)
	err := db.Where("id = ?", caseId).First(&c).Error
	return c, err
}

func (ca *caseDAO) GetPubByIDs(ctx context.Context, ids []int64) ([]PublishCase, error) {
	var c []PublishCase
	db := ca.db.WithContext(ctx)
	err := db.Where("id IN ?", ids).Find(&c).Error
	return c, err
}

func NewCaseDao(db *egorm.Component) CaseDAO {
	return &caseDAO{
		db:          db,
		listColumns: []string{"id", "labels", "status", "introduction", "title", "utime"},
		updateColumns: []string{
			"introduction", "labels", "title", "content",
			"github_repo", "gitee_repo", "keywords", "shorthand", "highlight",
			"guidance", "status", "utime", "biz", "biz_id"},
	}
}
