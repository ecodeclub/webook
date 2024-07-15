package dao

import (
	"context"
	"database/sql"
	"time"

	"github.com/ego-component/egorm"
	"gorm.io/gorm/clause"
)

// gpt扣分调用记录表
type GptCreditLog struct {
	Id     int64          `gorm:"primaryKey;autoIncrement;comment:积分流水表自增ID"`
	Tid    string         `gorm:"type:varchar(256);not null;comment:一次请求的Tid，可能有多次"`
	Uid    int64          `gorm:"not null;index:idx_user_id;comment:用户ID"`
	Biz    string         `gorm:"type:varchar(256);not null;comment:业务类型名"`
	Tokens int64          `gorm:"type:int;default:0;not null;comment:扣费token数"`
	Amount int64          `gorm:"type:int;default:0;not null;comment:具体扣费的换算的钱，分为单位"`
	Credit int64          `gorm:"type:int;default:0;not null;comment:具体扣费的积分"`
	Status uint8          `gorm:"type:tinyint unsigned;not null;default:0;comment:调用状态 0=进行中 1=成功, 2=失败"`
	Prompt sql.NullString `gorm:"type:text;comment:调用请求"`
	Answer sql.NullString `gorm:"type:text;comment:gpt的回答"`
	Ctime  int64
	Utime  int64
}

type GptLog struct {
	Id     int64          `gorm:"primaryKey;autoIncrement;comment:积分流水表自增ID"`
	Tid    string         `gorm:"type:varchar(256);not null;uniqueIndex:unq_tid;comment:一次请求的Tid只能有一次"`
	Uid    int64          `gorm:"not null;index:idx_user_id;comment:用户ID"`
	Biz    string         `gorm:"type:varchar(256);not null;comment:业务类型名"`
	Tokens int64          `gorm:"type:int;default:0;comment:扣费token数"`
	Amount int64          `gorm:"type:int;default:0;comment:具体扣费的换算的钱，分为单位"`
	Status uint8          `gorm:"type:tinyint unsigned;not null;default:1;comment:调用状态 1=成功, 2=失败"`
	Prompt sql.NullString `gorm:"type:text;comment:调用请求"`
	Answer sql.NullString `gorm:"type:text;comment:gpt的回答"`
	Ctime  int64
	Utime  int64
}

type GPTLogDAO interface {
	SaveCreditLog(ctx context.Context, GPTDeductLog GptCreditLog) (int64, error)
	SaveLog(ctx context.Context, GPTLog GptLog) (int64, error)
	FirstCreditLog(ctx context.Context, id int64) (*GptCreditLog, error)
	FirstLog(ctx context.Context, id int64) (*GptLog, error)
}

type gptLogDAO struct {
	db *egorm.Component
}

func NewGPTLogDAO(db *egorm.Component) GPTLogDAO {
	return &gptLogDAO{
		db: db,
	}
}

func (g *gptLogDAO) FirstCreditLog(ctx context.Context, id int64) (*GptCreditLog, error) {
	logModel := &GptCreditLog{}
	err := g.db.WithContext(ctx).Model(&GptCreditLog{}).Where("id = ?", id).First(logModel).Error
	return logModel, err
}

func (g *gptLogDAO) FirstLog(ctx context.Context, id int64) (*GptLog, error) {
	logModel := &GptLog{}
	err := g.db.WithContext(ctx).Model(&GptLog{}).Where("id = ?", id).First(logModel).Error
	return logModel, err
}

func (g *gptLogDAO) SaveCreditLog(ctx context.Context, gptLog GptCreditLog) (int64, error) {
	now := time.Now().UnixMilli()
	gptLog.Ctime = now
	gptLog.Utime = now
	err := g.db.WithContext(ctx).Model(&GptCreditLog{}).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{"status", "utime"}),
		}).Create(&gptLog).Error
	return gptLog.Id, err
}

func (g *gptLogDAO) SaveLog(ctx context.Context, gptLog GptLog) (int64, error) {
	now := time.Now().UnixMilli()
	gptLog.Ctime = now
	gptLog.Utime = now
	err := g.db.WithContext(ctx).Model(&GptLog{}).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{"status", "utime"}),
		}).Create(&gptLog).Error
	return gptLog.Id, err
}
