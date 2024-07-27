package dao

import (
	"github.com/ego-component/egorm"
)

func InitTables(db *egorm.Component) error {
	// 注册回掉
	err := db.Use(&UserPlugin{})
	if err != nil {
		return err
	}
	return db.AutoMigrate(
		&User{},
		&UsersIelts{},
	)
}

type UsersIelts User

func (u *UsersIelts) TableName() string {
	return "users_ielts"
}
