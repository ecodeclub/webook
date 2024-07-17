package dao

import (
	"context"
	"time"

	"github.com/ego-component/egorm"
	"gorm.io/gorm/clause"
)

// LLMCredit llm 扣分调用记录表
type LLMCredit struct {
	Id     int64  `gorm:"primaryKey;autoIncrement;comment:积分流水表自增ID"`
	Tid    string `gorm:"type:varchar(256);not null;comment:一次请求的Tid，可能有多次"`
	Uid    int64  `gorm:"not null;index:idx_user_id;comment:用户ID"`
	Biz    string `gorm:"type:varchar(256);not null;comment:业务类型名"`
	Amount int64  `gorm:"type:int;default:0;not null;comment:具体扣费的换算的钱，分为单位"`
	Status uint8  `gorm:"type:tinyint unsigned;not null;default:0;comment:调用状态 0=进行中 1=成功, 2=失败"`
	Ctime  int64
	Utime  int64
}

func (l LLMCredit) TableName() string {
	return "llm_credits"
}

type LLMCreditDAO interface {
	SaveCredit(ctx context.Context, l LLMCredit) (int64, error)
}

type GORMLLMCreditDAO struct {
	db *egorm.Component
}

func NewLLMCreditLogDAO(db *egorm.Component) LLMCreditDAO {
	return &GORMLLMCreditDAO{
		db: db,
	}
}

func (g *GORMLLMCreditDAO) SaveCredit(ctx context.Context, l LLMCredit) (int64, error) {
	now := time.Now().UnixMilli()
	l.Ctime = now
	l.Utime = now
	err := g.db.WithContext(ctx).Model(&LLMCredit{}).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{"status", "utime"}),
		}).Create(&l).Error
	return l.Id, err
}
