package dao

import (
	"context"
	"time"

	"github.com/ego-component/egorm"
	"gorm.io/gorm/clause"
)

type CompanyDAO interface {
	Save(ctx context.Context, c Company) (int64, error)
	FindById(ctx context.Context, id int64) (Company, error)
	FindByIds(ctx context.Context, ids []int64) ([]Company, error)
	List(ctx context.Context, offset int, limit int) ([]Company, error)
	Count(ctx context.Context) (int64, error)
	DeleteById(ctx context.Context, id int64) error
}

type GORMCompanyDAO struct {
	db *egorm.Component
}

func NewGORMCompanyDAO(db *egorm.Component) CompanyDAO {
	return &GORMCompanyDAO{
		db: db,
	}
}

func (c *GORMCompanyDAO) Save(ctx context.Context, company Company) (int64, error) {
	now := time.Now().UnixMilli()
	company.Utime = now
	company.Ctime = now
	err := c.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "utime"}),
	}).Create(&company).Error
	return company.Id, err

}

func (c *GORMCompanyDAO) FindById(ctx context.Context, id int64) (Company, error) {
	var company Company
	err := c.db.WithContext(ctx).Where("id = ?", id).First(&company).Error
	return company, err
}

func (c *GORMCompanyDAO) FindByIds(ctx context.Context, ids []int64) ([]Company, error) {
	var companies []Company
	err := c.db.WithContext(ctx).Where("id IN ?", ids).Find(&companies).Error
	return companies, err
}

func (c *GORMCompanyDAO) List(ctx context.Context, offset int, limit int) ([]Company, error) {
	var companies []Company
	err := c.db.WithContext(ctx).Offset(offset).Limit(limit).Order("utime DESC").Find(&companies).Error
	return companies, err
}

func (c *GORMCompanyDAO) Count(ctx context.Context) (int64, error) {
	var count int64
	err := c.db.WithContext(ctx).Model(&Company{}).Count(&count).Error
	return count, err
}

func (c *GORMCompanyDAO) DeleteById(ctx context.Context, id int64) error {
	return c.db.WithContext(ctx).Where("id = ?", id).Delete(&Company{}).Error
}

type Company struct {
	Id   int64  `gorm:"primaryKey,autoIncrement"`
	Name string `gorm:"type:varchar(256);not null"`
	// 创建时间
	Ctime int64
	// 更新时间
	Utime int64
}
