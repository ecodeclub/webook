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
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/marketing/internal/domain"
	"github.com/ego-component/egorm"
	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

var (
	ErrRedemptionNotFound = gorm.ErrRecordNotFound
	ErrRedemptionCodeUsed = errors.New("兑换码已使用")
)

type MarketingDAO interface {
	CreateRedemptionCodes(ctx context.Context, oid int64, code []RedemptionCode) ([]int64, error)
	FindRedemptionCodeByCode(ctx context.Context, code string) (RedemptionCode, error)
	SetUnusedRedemptionCodeStatusUsed(ctx context.Context, uid int64, code string) (RedemptionCode, error)
	CountRedemptionCodes(ctx context.Context, uid int64) (int64, error)
	FindRedemptionCodesByUID(ctx context.Context, uid int64, offset int, limit int) ([]RedemptionCode, error)
}

type gormMarketingDAO struct {
	db *egorm.Component
}

func NewGORMMarketingDAO(db *egorm.Component) MarketingDAO {
	return &gormMarketingDAO{db: db}
}

func (g *gormMarketingDAO) CreateRedemptionCodes(ctx context.Context, oid int64, codes []RedemptionCode) ([]int64, error) {
	now := time.Now().UnixMilli()
	for i := range codes {
		codes[i].Ctime, codes[i].Utime = now, now
	}
	err := g.db.WithContext(ctx).Transaction(func(tx *egorm.Component) error {
		if err := tx.Create(&codes).Error; err != nil {
			return fmt.Errorf("创建兑换码主记录失败: %w", err)
		}
		var l GenerateLog
		l.OrderId = oid
		l.CodeCount = int64(len(codes))
		l.Ctime = now
		l.Utime = now
		return tx.Create(&l).Error
	})
	if err != nil {
		if g.isMySQLUniqueIndexError(err) {
			return g.getRedemptionCodeIDsByOrderID(ctx, oid)
		}
		return nil, err
	}
	return slice.Map(codes, func(idx int, src RedemptionCode) int64 {
		return src.Id
	}), nil
}

func (g *gormMarketingDAO) isMySQLUniqueIndexError(err error) bool {
	var me *mysql.MySQLError
	if errors.As(err, &me) {
		const uniqueIndexErrNo uint16 = 1062
		if me.Number == uniqueIndexErrNo {
			return true
		}
	}
	return false
}

func (g *gormMarketingDAO) getRedemptionCodeIDsByOrderID(ctx context.Context, oid int64) ([]int64, error) {
	var res []int64
	err := g.db.WithContext(ctx).Model(&RedemptionCode{}).
		Select("id").Find(&res, "order_id", oid).Error
	return res, err
}

func (g *gormMarketingDAO) FindRedemptionCodeByCode(ctx context.Context, code string) (RedemptionCode, error) {
	var res RedemptionCode
	err := g.db.WithContext(ctx).First(&res, "code = ?", code).Error
	return res, err
}

func (g *gormMarketingDAO) SetUnusedRedemptionCodeStatusUsed(ctx context.Context, uid int64, code string) (RedemptionCode, error) {
	now := time.Now().UnixMilli()
	var c RedemptionCode
	err := g.db.WithContext(ctx).Transaction(func(tx *egorm.Component) error {

		updateResult := tx.Model(&c).Where("Code = ? AND Status = ?", code, domain.RedemptionCodeStatusUnused.ToUint8()).
			Updates(map[string]any{
				"Status": domain.RedemptionCodeStatusUsed.ToUint8(),
				"Utime":  now,
			})
		if updateResult.Error != nil {
			return updateResult.Error
		}

		if err := tx.Where("Code = ?", code).First(&c).Error; err != nil {
			return err
		}

		if updateResult.RowsAffected == 0 {
			return fmt.Errorf("%w: %s", ErrRedemptionCodeUsed, code)
		}

		l := RedeemLog{
			RId:        c.Id,
			RedeemerId: uid,
			Code:       code,
			OwnerId:    c.OwnerId,
			Ctime:      now,
			Utime:      now,
		}
		if err := tx.Create(&l).Error; err != nil {
			if g.isMySQLUniqueIndexError(err) {
				return fmt.Errorf("%w: %s", ErrRedemptionCodeUsed, code)
			}
			return err
		}
		return nil
	})
	if err != nil {
		return RedemptionCode{}, err
	}
	return c, err
}

func (g *gormMarketingDAO) CountRedemptionCodes(ctx context.Context, uid int64) (int64, error) {
	var count int64
	err := g.db.WithContext(ctx).Model(&RedemptionCode{}).Where("owner_id = ?", uid).
		Select("COUNT(id)").Count(&count).Error
	return count, err
}

func (g *gormMarketingDAO) FindRedemptionCodesByUID(ctx context.Context, uid int64, offset int, limit int) ([]RedemptionCode, error) {
	var res []RedemptionCode
	err := g.db.WithContext(ctx).Model(&RedemptionCode{}).Order("Utime DESC, id ASC").
		Offset(offset).Limit(limit).Find(&res, "owner_id = ?", uid).Error
	return res, err
}

type RedemptionCode struct {
	Id       int64          `gorm:"primaryKey;autoIncrement;comment:兑换码自增ID"`
	OwnerId  int64          `gorm:"not null;index:idx_owner_id;comment:所有者ID"`
	OrderId  int64          `gorm:"not null;index:idx_order_id;comment:订单自增ID"`
	SPUID    int64          `gorm:"column:spu_id;not null;index:idx_spu_id;comment:订单项对应的SPU自增ID"`
	SPUType  string         `gorm:"column:spu_type;type:varchar(255);not null;comment:订单项对应的SPU的类型, 仅内部使用member/project"`
	SKUAttrs sql.NullString `gorm:"comment:商品销售属性,JSON格式"`
	Code     string         `gorm:"type:varchar(255);not null;uniqueIndex:uniq_code;comment:兑换码"`
	Status   uint8          `gorm:"type:tinyint unsigned;not null;default:1;comment:使用状态 1=未使用 2=已使用"`
	Ctime    int64
	Utime    int64
}

type GenerateLog struct {
	Id        int64 `gorm:"primaryKey;autoIncrement;comment:兑换码兑换记录ID"`
	OrderId   int64 `gorm:"not null;uniqueIndex:idx_order_id_code_count;comment:订单ID"`
	CodeCount int64 `gorm:"not null;uniqueIndex:idx_order_id_code_count;comment:订单中包含的兑换码个数"`
	Ctime     int64
	Utime     int64
}

type RedeemLog struct {
	Id         int64  `gorm:"primaryKey;autoIncrement;comment:兑换码兑换记录ID"`
	RId        int64  `gorm:"not null;uniqueIndex:uniq_redemption_code_id;comment:兑换码记录自增ID"`
	RedeemerId int64  `gorm:"not null;index:idx_redeemer_id;comment:兑换者ID"`
	Code       string `gorm:"type:varchar(255);not null;uniqueIndex:uniq_code;comment:兑换码"`
	OwnerId    int64  `gorm:"not null;index:idx_owner_id;comment:所有者ID"`
	Ctime      int64
	Utime      int64
}
