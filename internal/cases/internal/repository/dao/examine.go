package dao

import (
	"context"
	"github.com/ego-component/egorm"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

var ErrRecordNotFound = gorm.ErrRecordNotFound

type ExamineDAO interface {
	SaveResult(ctx context.Context, record CaseExamineRecord) error
	GetResultByUidAndCid(ctx context.Context, uid int64, cid int64) (CaseResult, error)
	GetResultByUidAndCids(ctx context.Context, uid int64, cids []int64) ([]CaseResult, error)
}

type GORMExamineDAO struct {
	db *egorm.Component
}

func NewGORMExamineDAO(db *egorm.Component) ExamineDAO {
	return &GORMExamineDAO{
		db: db,
	}
}

func (dao *GORMExamineDAO) SaveResult(ctx context.Context, record CaseExamineRecord) error {
	now := time.Now().UnixMilli()
	return dao.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		record.Ctime = now
		record.Utime = now
		err := tx.Create(&record).Error
		if err != nil {
			return err
		}
		return tx.Clauses(clause.OnConflict{
			// 如果有记录了，就更新结果和更新时间
			DoUpdates: clause.AssignmentColumns([]string{
				"result", "utime",
			}),
		}).Create(&CaseResult{
			Uid:    record.Uid,
			Cid:    record.Cid,
			Result: record.Result,
			Ctime:  now,
			Utime:  now,
		}).Error
	})
}

func (dao *GORMExamineDAO) GetResultByUidAndCid(ctx context.Context, uid int64, cid int64) (CaseResult, error) {
	var res CaseResult
	err := dao.db.WithContext(ctx).Where("uid = ? AND cid = ?", uid, cid).First(&res).Error
	return res, err
}

func (dao *GORMExamineDAO) GetResultByUidAndCids(ctx context.Context, uid int64, cids []int64) ([]CaseResult, error) {
	var res []CaseResult
	err := dao.db.WithContext(ctx).Where("uid = ? AND cid IN ?", uid, cids).Find(&res).Error
	return res, err
}
