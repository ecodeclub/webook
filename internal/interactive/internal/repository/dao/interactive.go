// Copyright 2023 ecodeclub
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dao

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ego-component/egorm"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrDeleteOtherCollection = errors.New("删除非本人的收藏夹")

type InteractiveDAO interface {
	IncrViewCnt(ctx context.Context, biz string, bizId int64) error
	LikeToggle(ctx context.Context, biz string, id int64, uid int64) error
	CollectToggle(ctx context.Context, cb UserCollectionBiz) error
	GetLikeInfo(ctx context.Context,
		biz string, id int64, uid int64) (UserLikeBiz, error)
	GetCollectInfo(ctx context.Context,
		biz string, id int64, uid int64) (UserCollectionBiz, error)
	Get(ctx context.Context, biz string, id int64) (Interactive, error)
	GetByIds(ctx context.Context, biz string, ids []int64) ([]Interactive, error)
	GetUserLikes(ctx context.Context, uid int64, biz string, ids []int64) ([]UserLikeBiz, error)

	GetUserCollects(ctx context.Context, uid int64, biz string, ids []int64) ([]UserCollectionBiz, error)

	// 创建收藏夹
	SaveCollection(ctx context.Context, collection Collection) (int64, error)
	// 删除收藏夹
	DeleteCollection(ctx context.Context, uid, collectionId int64) error
	// 收藏夹列表
	CollectionList(ctx context.Context, uid int64, offset, limit int) ([]Collection, error)
	// 收藏夹下的收藏内容带分页
	CollectionInfoWithPage(ctx context.Context, uid, collectionId int64, offset, limit int) ([]UserCollectionBiz, error)
	// 收藏夹下的所有内容
	CollectionInfo(ctx context.Context, collectionId int64) ([]UserCollectionBiz, error)
	// 收藏转移
	MoveCollection(ctx context.Context, biz string, bizid, uid, collectionId int64) error
	// 减少计数
	DecrCollectCount(ctx context.Context, biz string, bizid int64) error
}

type GORMInteractiveDAO struct {
	db *egorm.Component
}

func (g *GORMInteractiveDAO) DecrCollectCount(ctx context.Context, biz string, bizid int64) error {
	return g.db.WithContext(ctx).
		Model(&Interactive{}).
		Where("biz = ? AND biz_id = ? and collect_cnt > 0", biz, bizid).
		Update("collect_cnt", gorm.Expr("`collect_cnt` - 1")).Error
}

func NewInteractiveDAO(db *egorm.Component) *GORMInteractiveDAO {
	return &GORMInteractiveDAO{
		db: db,
	}
}

func (g *GORMInteractiveDAO) GetUserLikes(ctx context.Context, uid int64, biz string, ids []int64) ([]UserLikeBiz, error) {
	var likes []UserLikeBiz
	err := g.db.WithContext(ctx).
		Model(&UserLikeBiz{}).
		Where("biz = ? AND biz_id in ? and uid = ?", biz, ids, uid).Scan(&likes).Error
	return likes, err
}

func (g *GORMInteractiveDAO) GetUserCollects(ctx context.Context, uid int64, biz string, ids []int64) ([]UserCollectionBiz, error) {
	var collects []UserCollectionBiz
	err := g.db.WithContext(ctx).
		Model(&UserCollectionBiz{}).
		Where("biz = ? AND biz_id in ? and uid = ?", biz, ids, uid).Scan(&collects).Error
	return collects, err
}

func (g *GORMInteractiveDAO) SaveCollection(ctx context.Context, collection Collection) (int64, error) {
	now := time.Now()
	ctime := now.UnixMilli()
	utime := now.UnixMilli()
	collection.Utime = utime
	collection.Ctime = ctime
	err := g.db.WithContext(ctx).Clauses(
		clause.OnConflict{
			Columns: []clause.Column{
				{
					Name: "id",
				},
			},
			DoUpdates: clause.Assignments(map[string]any{
				"name":  collection.Name,
				"utime": collection.Utime,
			}),
		},
	).Create(&collection).Error
	if err != nil {
		return 0, err
	}
	return collection.Id, nil
}

func (g *GORMInteractiveDAO) DeleteCollection(ctx context.Context, uid, collectionId int64) error {
	return g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除收藏夹
		res := tx.Model(&Collection{}).Where("uid = ? AND id = ?", uid, collectionId).Delete(&Collection{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected < 1 {
			return fmt.Errorf("%w", ErrDeleteOtherCollection)
		}
		// 删除收藏内容
		return tx.Model(&UserCollectionBiz{}).Where("cid = ? AND uid = ?", collectionId, uid).Delete(&UserCollectionBiz{}).Error
	})
}

func (g *GORMInteractiveDAO) CollectionList(ctx context.Context, uid int64, offset, limit int) ([]Collection, error) {
	var collections []Collection
	err := g.db.WithContext(ctx).
		Model(&Collection{}).
		Where("uid = ?", uid).
		Order("id DESC").
		Limit(limit).
		Offset(offset).Scan(&collections).Error
	return collections, err
}

func (g *GORMInteractiveDAO) CollectionInfoWithPage(ctx context.Context, uid, collectionId int64, offset, limit int) ([]UserCollectionBiz, error) {
	records := make([]UserCollectionBiz, 0, 32)
	err := g.db.WithContext(ctx).
		Model(&UserCollectionBiz{}).
		Where("cid = ? AND uid = ? ", collectionId, uid).
		Order("id DESC").
		Offset(offset).
		Limit(limit).
		Find(&records).Error
	return records, err
}

func (g *GORMInteractiveDAO) CollectionInfo(ctx context.Context, collectionId int64) ([]UserCollectionBiz, error) {
	records := make([]UserCollectionBiz, 0, 32)
	err := g.db.WithContext(ctx).Where("cid = ?", collectionId).Find(&records).Error
	return records, err
}

func (g *GORMInteractiveDAO) MoveCollection(ctx context.Context, biz string, bizid, uid, collectionId int64) error {
	err := g.db.WithContext(ctx).
		Model(&UserCollectionBiz{}).
		Where("biz = ? AND biz_id = ? AND uid = ?", biz, bizid, uid).
		Update("cid", collectionId).Error
	return err
}

func (g *GORMInteractiveDAO) LikeToggle(ctx context.Context, biz string, id int64, uid int64) error {
	return g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.
			Where("biz = ? AND biz_id = ? AND uid = ?", biz, id, uid).
			First(&UserLikeBiz{}).Error
		switch {
		case err == nil:
			return g.deleteLikeInfo(tx, biz, id, uid)
		case errors.Is(err, gorm.ErrRecordNotFound):
			return g.insertLikeInfo(tx, biz, id, uid)
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

func (g *GORMInteractiveDAO) CollectToggle(ctx context.Context, cb UserCollectionBiz) error {
	return g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.
			Where("biz = ? AND biz_id = ? AND uid = ?", cb.Biz, cb.BizId, cb.Uid).
			First(&UserCollectionBiz{}).Error
		switch {
		case err == nil:
			return g.deleteCollectionInfo(tx, cb.Biz, cb.BizId, cb.Uid)
		case errors.Is(err, gorm.ErrRecordNotFound):
			return g.insertCollectionBiz(tx, cb)
		default:
			return err
		}
	})
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
