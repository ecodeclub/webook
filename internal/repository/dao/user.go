package dao

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

var (
	ErrUserDuplicate = errors.New("邮箱已注册，请登录！")
)

type UserDAO interface {
	Insert(ctx context.Context, u User) error
	Update(ctx context.Context, u User) error
	FindByEmail(ctx context.Context, email string) (User, error)
}

type User struct {
	Id          int64  `gorm:"primaryKey;autoIncrement"`
	Email       string `gorm:"unique"`
	EmailVerify sql.NullByte
	Password    string
	CreateTime  int64
	UpdateTime  int64
}

type GormUserDAO struct {
	db *gorm.DB
}

func NewUserInfoDAO(db *gorm.DB) UserDAO {
	return &GormUserDAO{
		db: db,
	}
}

func InitTables(db *gorm.DB) error {
	return db.AutoMigrate(&User{})
}

func (dao *GormUserDAO) Insert(ctx context.Context, u User) error {
	now := time.Now().UnixMilli()
	u.CreateTime = now
	u.UpdateTime = now

	err := dao.db.WithContext(ctx).Create(&u).Error
	if e, ok := err.(*mysql.MySQLError); ok {
		const uniqueIndexErr uint16 = 1062
		// 检查错误编号是否表示唯一索引冲突
		if e.Number == uniqueIndexErr {
			return ErrUserDuplicate
		}
	}
	return err
}

func (dao *GormUserDAO) Update(ctx context.Context, u User) error {
	return dao.db.WithContext(ctx).Save(&u).Error
}

func (dao *GormUserDAO) FindByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := dao.db.WithContext(ctx).First(&u, "email = ?", email).Error
	return u, err
}
