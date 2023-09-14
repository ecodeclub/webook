package dao

import (
	"context"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

var (
	ErrUserDuplicate = errors.New("邮箱已注册，请登录！")
	ErrDataNotFound  = gorm.ErrRecordNotFound
)

type UserDAO interface {
	Insert(ctx context.Context, u User) error
	UpdateEmailVerifiedByEmail(ctx context.Context, email string) error
	FindByEmail(ctx context.Context, email string) (User, error)
}

type User struct {
	Id            int64  `gorm:"primaryKey;autoIncrement"`
	Email         string `gorm:"size:256;unique;comment:邮箱"`
	Password      string `gorm:"size:128;comment:密码"`
	EmailVerified bool   `gorm:"comment:邮箱已验证"` // 邮箱验证最多允许更换，不应该允许取消验证，因此使用bool。若需求有变可以考虑定义uint8
	CreateTime    int64  `gorm:"comment:创建时间(milli)"`
	UpdateTime    int64  `gorm:"comment:更新时间(milli)"`
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

func (dao *GormUserDAO) UpdateEmailVerifiedByEmail(ctx context.Context, email string) error {
	now := time.Now().UnixMilli()
	return dao.db.WithContext(ctx).Model(&User{}).
		Where(&User{Email: email}, "Email").
		Updates(&User{EmailVerified: true, UpdateTime: now}).Error
}

func (dao *GormUserDAO) FindByEmail(ctx context.Context, email string) (User, error) {
	var u User
	return u, dao.db.WithContext(ctx).Model(&User{}).
		Where(&User{Email: email}, "Email"). // 必需根据Email查询
		Take(&u).Error                       // Take 不需要排序
}
