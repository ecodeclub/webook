package dao

import (
	"context"
	"errors"
	"time"

	"github.com/ego-component/egorm"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type InteractiveDAO interface {
	IncrViewCnt(ctx context.Context, biz string, bizId int64) error
	LikeToggle(ctx context.Context, biz string, id int64, uid int64) error
	CollectionToggle(ctx context.Context, cb UserCollectionBiz) error
	GetLikeInfo(ctx context.Context,
		biz string, id int64, uid int64) (UserLikeBiz, error)
	GetCollectInfo(ctx context.Context,
		biz string, id int64, uid int64) (UserCollectionBiz, error)
	Get(ctx context.Context, biz string, id int64) (Interactive, error)
	GetByIds(ctx context.Context, biz string, ids []int64) ([]Interactive, error)
}

type GORMInteractiveDAO struct {
	db *egorm.Component
}

func NewInteractiveDAO(db *egorm.Component) *GORMInteractiveDAO {
	return &GORMInteractiveDAO{
		db: db,
	}
}

func (g *GORMInteractiveDAO) LikeToggle(ctx context.Context, biz string, id int64, uid int64) error {
	return g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var res UserLikeBiz
		err := tx.Where("biz = ? AND biz_id = ? AND uid = ?", biz, id, uid).
			First(&res).Error
		switch err {
		case nil:
			return g.deleteLikeInfo(tx, biz, id, uid)
		case gorm.ErrRecordNotFound:
			return g.insertLikeInfo(tx, biz, id, uid)
		default:
			return err
		}
	})
}

func (g *GORMInteractiveDAO) CollectionToggle(ctx context.Context, cb UserCollectionBiz) error {
	return g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var res UserCollectionBiz
		err := tx.Where("biz = ? AND biz_id = ? AND uid = ?", cb.Biz, cb.BizId, cb.Uid).
			First(&res).Error
		switch err {
		case nil:
			return g.deleteCollectionInfo(tx, cb.Biz, cb.BizId, cb.Uid)
		case gorm.ErrRecordNotFound:
			return g.insertCollectionBiz(tx, cb)
		default:
			return err
		}
	})
}
func (g *GORMInteractiveDAO) insertLikeInfo(tx *gorm.DB, biz string, id int64, uid int64) error {
	now := time.Now().UnixMilli()
	err := tx.Create(&UserLikeBiz{
		Uid:   uid,
		Biz:   biz,
		BizId: id,
		Utime: now,
		Ctime: now,
	}).Error
	if err != nil {
		return err
	}
	return tx.Clauses(clause.OnConflict{
		DoUpdates: clause.Assignments(map[string]any{
			"like_cnt": gorm.Expr("`like_cnt` + 1"),
			"utime":    now,
		}),
	}).Create(&Interactive{
		Biz:     biz,
		BizId:   id,
		LikeCnt: 1,
		Ctime:   now,
		Utime:   now,
	}).Error
}

func (g *GORMInteractiveDAO) insertCollectionBiz(tx *gorm.DB, cb UserCollectionBiz) error {
	now := time.Now().UnixMilli()
	cb.Ctime = now
	cb.Utime = now
	err := tx.Create(&cb).Error
	if err != nil {
		return err
	}
	return tx.Clauses(clause.OnConflict{
		DoUpdates: clause.Assignments(map[string]any{
			"collect_cnt": gorm.Expr("`collect_cnt` + 1"),
			"utime":       now,
		}),
	}).Create(&Interactive{
		Biz:        cb.Biz,
		BizId:      cb.BizId,
		CollectCnt: 1,
		Ctime:      now,
		Utime:      now,
	}).Error

}

func (g *GORMInteractiveDAO) deleteLikeInfo(tx *gorm.DB, biz string, id int64, uid int64) error {
	now := time.Now().UnixMilli()
	res := tx.Model(&UserLikeBiz{}).
		Where("uid=? AND biz_id = ? AND biz=?", uid, id, biz).
		Delete(&UserLikeBiz{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected < 1 {
		return nil
	}
	return tx.Model(&Interactive{}).
		Where("biz =? AND biz_id=?", biz, id).
		Updates(map[string]any{
			"like_cnt": gorm.Expr("`like_cnt` - 1"),
			"utime":    now,
		}).Error
}

func (g *GORMInteractiveDAO) deleteCollectionInfo(tx *gorm.DB, biz string, id int64, uid int64) error {
	now := time.Now().UnixMilli()
	res := tx.Model(&UserCollectionBiz{}).
		Where("uid=? AND biz_id = ? AND biz=?", uid, id, biz).
		Delete(&UserCollectionBiz{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected < 1 {
		return nil
	}
	return tx.Model(&Interactive{}).
		Where("biz =? AND biz_id=?", biz, id).
		Updates(map[string]any{
			"collect_cnt": gorm.Expr("`collect_cnt` - 1"),
			"utime":       now,
		}).Error
}

func (g *GORMInteractiveDAO) IncrViewCnt(ctx context.Context, biz string, bizId int64) error {
	now := time.Now().UnixMilli()
	return g.db.WithContext(ctx).Clauses(clause.OnConflict{
		DoUpdates: clause.Assignments(map[string]any{
			"view_cnt": gorm.Expr("`view_cnt` + 1"),
			"utime":    now,
		}),
	}).Create(&Interactive{
		Biz:     biz,
		BizId:   bizId,
		ViewCnt: 1,
		Ctime:   now,
		Utime:   now,
	}).Error
}

func (g *GORMInteractiveDAO) GetLikeInfo(ctx context.Context, biz string, id int64, uid int64) (UserLikeBiz, error) {
	var res UserLikeBiz
	err := g.db.WithContext(ctx).
		Where("biz = ? AND biz_id = ? AND uid = ?",
			biz, id, uid).
		First(&res).Error
	return res, err
}

func (g *GORMInteractiveDAO) GetCollectInfo(ctx context.Context, biz string, id int64, uid int64) (UserCollectionBiz, error) {
	var res UserCollectionBiz
	err := g.db.WithContext(ctx).
		Where("biz = ? AND biz_id = ? AND uid = ?", biz, id, uid).
		First(&res).Error
	return res, err
}

func (g *GORMInteractiveDAO) Get(ctx context.Context, biz string, id int64) (Interactive, error) {
	var res Interactive
	err := g.db.WithContext(ctx).
		Where("biz = ? AND biz_id = ?", biz, id).
		First(&res).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Interactive{}, ErrRecordNotFound
	}
	return res, err
}

func (g *GORMInteractiveDAO) GetByIds(ctx context.Context, biz string, ids []int64) ([]Interactive, error) {
	var res []Interactive
	err := g.db.WithContext(ctx).
		Where("biz = ? AND biz_id IN ?", biz, ids).
		Order("biz_id desc").
		Find(&res).Error
	return res, err
}
