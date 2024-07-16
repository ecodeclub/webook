package dao

import (
	"context"
	"time"

	"github.com/ego-component/egorm"
	"gorm.io/gorm/clause"
)

// GPTCredit gpt扣分调用记录表
type GPTCredit struct {
	Id     int64  `gorm:"primaryKey;autoIncrement;comment:积分流水表自增ID"`
	Tid    string `gorm:"type:varchar(256);not null;comment:一次请求的Tid，可能有多次"`
	Uid    int64  `gorm:"not null;index:idx_user_id;comment:用户ID"`
	Biz    string `gorm:"type:varchar(256);not null;comment:业务类型名"`
	Amount int64  `gorm:"type:int;default:0;not null;comment:具体扣费的换算的钱，分为单位"`
	Status uint8  `gorm:"type:tinyint unsigned;not null;default:0;comment:调用状态 0=进行中 1=成功, 2=失败"`
	Ctime  int64
	Utime  int64
}

func (l GPTCredit) TableName() string {
	return "gpt_credits"
}

type GPTCreditDAO interface {
	SaveCredit(ctx context.Context, GPTDeductLog GPTCredit) (int64, error)
}

type GORMGPTCreditDAO struct {
	db *egorm.Component
}

func NewGPTCreditLogDAO(db *egorm.Component) GPTCreditDAO {
	return &GORMGPTCreditDAO{
		db: db,
	}
}

func (g *GORMGPTCreditDAO) SaveCredit(ctx context.Context, gptLog GPTCredit) (int64, error) {
	now := time.Now().UnixMilli()
	gptLog.Ctime = now
	gptLog.Utime = now
	err := g.db.WithContext(ctx).Model(&GPTCredit{}).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{"status", "utime"}),
		}).Create(&gptLog).Error
	return gptLog.Id, err
}
