package dao

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/ego-component/egorm"
	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

// ErrDataNotFound 通用的数据没找到
var ErrDataNotFound = gorm.ErrRecordNotFound

// ErrUserDuplicate 这个算是 user 专属的
var ErrUserDuplicate = errors.New("用户已经注册")

//go:generate mockgen -source=./user.go -package=daomocks -destination=mocks/user.mock.go UserDAO
type UserDAO interface {
	Insert(ctx context.Context, u User) (int64, error)
	UpdateNonZeroFields(ctx context.Context, u User) error
	FindByWechat(ctx context.Context, unionId string) (User, error)
	FindById(ctx context.Context, id int64) (User, error)
	FindByIds(ctx context.Context, ids []int64) ([]User, error)
}

type GORMUserDAO struct {
	db *egorm.Component
}

func NewGORMUserDAO(db *egorm.Component) UserDAO {
	return &GORMUserDAO{
		db: db,
	}
}

func (ud *GORMUserDAO) UpdateNonZeroFields(ctx context.Context, u User) error {
	return ud.db.WithContext(ctx).Updates(&u).Error
}

func (ud *GORMUserDAO) Insert(ctx context.Context, u User) (int64, error) {
	now := time.Now().UnixMilli()
	u.Ctime = now
	u.Utime = now
	err := ud.db.WithContext(ctx).Create(&u).Error
	if me, ok := err.(*mysql.MySQLError); ok {
		const uniqueIndexErrNo uint16 = 1062
		if me.Number == uniqueIndexErrNo {
			return 0, ErrUserDuplicate
		}
	}
	return u.Id, err
}

func (ud *GORMUserDAO) FindByWechat(ctx context.Context, unionId string) (User, error) {
	var u User
	err := ud.db.WithContext(ctx).First(&u, "wechat_union_id = ?", unionId).Error
	return u, err
}

func (ud *GORMUserDAO) FindById(ctx context.Context, id int64) (User, error) {
	var u User
	err := ud.db.WithContext(ctx).First(&u, "id = ?", id).Error
	return u, err
}

func (ud *GORMUserDAO) FindByIds(ctx context.Context, ids []int64) ([]User, error) {
	var us []User
	err := ud.db.WithContext(ctx).Find(&us, "id IN ?", ids).Error
	return us, err
}

type User struct {
	Id       int64 `gorm:"primaryKey,autoIncrement"`
	Nickname string
	Avatar   string
	SN       string `gorm:"type:varchar(256);unique"`
	// TODO 后面要考虑拆分出去作为单表了
	WechatOpenId     sql.NullString `gorm:"type:varchar(256);unique"`
	WechatUnionId    sql.NullString `gorm:"type:varchar(256);unique"`
	WechatMiniOpenId sql.NullString `gorm:"type:varchar(256);unique"`
	// 创建时间
	Ctime int64
	// 更新时间
	Utime int64
}
