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

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ekit/sqlx"
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
	CreateRedemptionCodes(ctx context.Context, codes []RedemptionCode) ([]int64, error)
	FindRedemptionCodeByCode(ctx context.Context, code string) (RedemptionCode, error)
	SetUnusedRedemptionCodeStatusUsed(ctx context.Context, uid int64, code string) (RedemptionCode, error)
	CountRedemptionCodes(ctx context.Context, uid int64) (int64, error)
	FindRedemptionCodesByUID(ctx context.Context, uid int64, offset int, limit int) ([]RedemptionCode, error)
	CreateInvitationCode(ctx context.Context, i InvitationCode) (int64, error)
	FindInvitationCodeByCode(ctx context.Context, code string) (InvitationCode, error)
	CreateInvitationRecord(ctx context.Context, ir InvitationRecord) (int64, error)
	FindInvitationRecord(ctx context.Context, inviterId, inviteeId int64, code string) (InvitationRecord, error)
}

type gormMarketingDAO struct {
	db *egorm.Component
}

func NewGORMMarketingDAO(db *egorm.Component) MarketingDAO {
	return &gormMarketingDAO{db: db}
}

func (g *gormMarketingDAO) CreateRedemptionCodes(ctx context.Context, codes []RedemptionCode) ([]int64, error) {
	now := time.Now().UnixMilli()
	var biz string
	var bizId int64
	for i := range codes {
		codes[i].Ctime, codes[i].Utime = now, now
		biz, bizId = codes[i].Biz, codes[i].BizId
	}
	err := g.db.WithContext(ctx).Transaction(func(tx *egorm.Component) error {
		if err := tx.Create(&codes).Error; err != nil {
			return fmt.Errorf("创建兑换码主记录失败: %w", err)
		}
		var l GenerateLog
		l.Biz = biz
		l.BizId = bizId
		l.CodeCount = int64(len(codes))
		l.Ctime = now
		l.Utime = now
		return tx.Create(&l).Error
	})
	if err != nil {
		if g.isMySQLUniqueIndexError(err) {
			return g.getRedemptionCodeIDsByBizID(ctx, bizId)
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

func (g *gormMarketingDAO) getRedemptionCodeIDsByBizID(ctx context.Context, bizId int64) ([]int64, error) {
	var res []int64
	err := g.db.WithContext(ctx).Model(&RedemptionCode{}).
		Select("id").Find(&res, "biz_id", bizId).Error
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
		updateResult := tx.Model(&c).Where("code = ? AND status = ?", code, domain.RedemptionCodeStatusUnused.ToUint8()).
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

func (g *gormMarketingDAO) CreateInvitationCode(ctx context.Context, i InvitationCode) (int64, error) {
	now := time.Now().UnixMilli()
	i.Ctime, i.Utime = now, now
	err := g.db.WithContext(ctx).Create(&i).Error
	return i.Id, err
}

func (g *gormMarketingDAO) FindInvitationCodeByCode(ctx context.Context, code string) (InvitationCode, error) {
	var ic InvitationCode
	err := g.db.WithContext(ctx).First(&ic, "code = ?", code).Error
	return ic, err
}

func (g *gormMarketingDAO) CreateInvitationRecord(ctx context.Context, ir InvitationRecord) (int64, error) {
	now := time.Now().UnixMilli()
	ir.Ctime, ir.Utime = now, now
	err := g.db.WithContext(ctx).Attrs(InvitationRecord{InviterId: ir.InviterId, InviteeId: ir.InviteeId, Code: ir.Code}).
		FirstOrCreate(&ir).Error
	return ir.Id, err
}

func (g *gormMarketingDAO) FindInvitationRecord(ctx context.Context, inviterId, inviteeId int64, code string) (InvitationRecord, error) {
	var res InvitationRecord
	err := g.db.WithContext(ctx).First(&res, "inviter_id = ? AND invitee_id = ? AND code = ?", inviterId, inviteeId, code).Error
	return res, err
}

type RedemptionCode struct {
	Id      int64                             `gorm:"primaryKey;autoIncrement;comment:兑换码自增ID"`
	OwnerId int64                             `gorm:"not null;index:idx_owner_id;comment:所有者ID"`
	Biz     string                            `gorm:"type:varchar(255);not null;index:idx_biz;comment:业务名,admin/order等"`
	BizId   int64                             `gorm:"not null;index:idx_biz_id;comment:业务唯一ID, order_id等"`
	Type    string                            `gorm:"not null;index:idx_type;comment:兑换码类型与SPU的Category1对应,member/project/interview等"`
	Attrs   sqlx.JsonColumn[domain.CodeAttrs] `gorm:"type:varchar(512);not null;comment:商品销售属性,JSON格式,根据Type来解析Attrs"`
	Code    string                            `gorm:"type:varchar(255);not null;uniqueIndex:uniq_code;comment:兑换码"`
	Status  uint8                             `gorm:"type:tinyint unsigned;not null;default:1;comment:使用状态 1=未使用 2=已使用"`
	Ctime   int64
	Utime   int64
}

type GenerateLog struct {
	Id        int64  `gorm:"primaryKey;autoIncrement;comment:兑换码兑换记录ID"`
	Biz       string `gorm:"type:varchar(255);not null;uniqueIndex:uniq_biz_biz_id_code_count;comment:此次生成兑换码的业务名,admin/order等"`
	BizId     int64  `gorm:"not null;uniqueIndex:uniq_biz_biz_id_code_count;comment:生成兑换码业务的唯一ID, order_id等"`
	CodeCount int64  `gorm:"not null;uniqueIndex:uniq_biz_biz_id_code_count;comment:此次活动中生成兑换码个数"`
	Ctime     int64
	Utime     int64
}

type RedeemLog struct {
	Id         int64  `gorm:"primaryKey;autoIncrement;comment:兑换码兑换记录ID"`
	RId        int64  `gorm:"not null;uniqueIndex:uniq_redemption_code_id;comment:兑换码记录自增ID"`
	RedeemerId int64  `gorm:"not null;index:idx_redeemer_id;comment:兑换者ID"`
	Code       string `gorm:"type:varchar(255);not null;uniqueIndex:uniq_code;comment:兑换码"`
	Ctime      int64
	Utime      int64
}

type InvitationCode struct {
	Id      int64  `gorm:"primaryKey;autoIncrement;comment:邀请码自增ID"`
	OwnerId int64  `gorm:"not null;index:idx_owner_id;comment:所有者ID"`
	Code    string `gorm:"type:varchar(255);not null;uniqueIndex:uniq_invitation_code;comment:邀请码"`
	Ctime   int64
	Utime   int64
}

type InvitationRecord struct {
	Id        int64                                         `gorm:"primaryKey;autoIncrement;comment:邀请记录自增ID"`
	InviterId int64                                         `gorm:"not null;index:idx_inviter_id;comment:邀请者ID"`
	InviteeId int64                                         `gorm:"not null;uniqueIndex:uniq_invitee_id;comment:被邀请者ID"`
	Code      string                                        `gorm:"type:varchar(255);not null;uniqueIndex:uniq_invitation_code;comment:邀请码"`
	Attrs     sqlx.JsonColumn[domain.InvitationRecordAttrs] `gorm:"type:varchar(512);not null;comment:邀请记录的其他属性Attrs"`
	Ctime     int64
	Utime     int64
}
