package dao

import "gorm.io/gorm"

type UserPlugin struct {
}

func (u *UserPlugin) Name() string {
	return "user"
}

func (u *UserPlugin) Initialize(db *gorm.DB) error {
	// 注册回掉
	insertBuilder, err := NewUserInsertCallBackBuilder(0, 2)
	if err != nil {
		panic(err)
	}
	shardingByApp := NewUserCallBackBuilder().Build()
	err = db.Callback().Query().Before("*").Register("user_query", shardingByApp)
	if err != nil {
		return err
	}
	err = db.Callback().Delete().Before("*").Register("user_delete", shardingByApp)
	if err != nil {
		return err
	}
	err = db.Callback().Create().Before("*").Register("user_create", insertBuilder.Build())
	if err != nil {
		return err
	}
	return db.Callback().Update().Before("*").Register("user_update", shardingByApp)
}
