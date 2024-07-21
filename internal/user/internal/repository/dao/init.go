package dao

import (
	"github.com/ego-component/egorm"
)

func InitTables(db *egorm.Component) error {
	// 注册回掉
	insertBuilder, err := NewUserInsertCallBackBuilder(0, 2)
	if err != nil {
		panic(err)
	}
	err = db.Callback().Query().Before("*").Register("user_query", NewUserCallBackBuilder().Build())
	if err != nil {
		panic(err)
	}
	err = db.Callback().Delete().Before("*").Register("user_delete", NewUserCallBackBuilder().Build())
	if err != nil {
		panic(err)
	}
	err = db.Callback().Create().Before("*").Register("user_create", insertBuilder.Build())
	if err != nil {
		panic(err)
	}
	err = db.Callback().Update().Before("*").Register("user_update", NewUserCallBackBuilder().Build())
	if err != nil {
		panic(err)
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
