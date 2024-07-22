package dao

import (
	"context"
	"fmt"

	"github.com/ecodeclub/webook/internal/pkg/ectx"

	"github.com/ecodeclub/webook/internal/pkg/snowflake"
	"github.com/gotomicro/ego/core/elog"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const (
	uidCtxKey = "uid"
)

var appMap = map[uint]string{
	1: "ielts",
}
var (
	WebookApp       = uint(0)
	UserTableName   = "users"
	ErrUnknownAppid = errors.New("未知的appid")
)

type UserInsertCallBackBuilder struct {
	logger  *elog.Component
	idMaker snowflake.AppIDGenerator
}

func NewUserInsertCallBackBuilder(nodeid, apps uint) (*UserInsertCallBackBuilder, error) {
	idMaker, err := snowflake.NewMeoyingIDGenerator(nodeid, apps)
	if err != nil {
		return nil, err
	}
	return &UserInsertCallBackBuilder{
		logger:  elog.DefaultLogger,
		idMaker: idMaker,
	}, nil
}

func (u *UserInsertCallBackBuilder) Build() func(db *gorm.DB) {
	return func(db *gorm.DB) {
		table := db.Statement.Table
		if table == UserTableName {
			appid, ok := appId(db.Statement.Context)
			// 没设置就是默认的webook的appid
			if !ok {
				appid = WebookApp
			}
			id, err := u.idMaker.Generate(appid)
			if err != nil {
				u.logger.Error("获取雪花id失败", elog.FieldErr(err))
				return
			}
			// 修改表名
			tableName, err := tableNameFromAppId(appid)
			if err != nil {
				u.logger.Error("获取别名失败", elog.FieldErr(err))
				return
			}
			db.Statement.Table = tableName
			us, ok := db.Statement.Dest.(*User)
			if !ok {
				u.logger.Error("修改id失败", elog.FieldErr(err))
				return
			}
			if us.Id == 0 {
				us.Id = id.Int64()
			}
			db.Statement.Dest = us
		}
	}
}

// 除insert以外的语句
type UserCallBackBuilder struct {
	logger *elog.Component
}

func NewUserCallBackBuilder() *UserCallBackBuilder {
	return &UserCallBackBuilder{
		logger: elog.DefaultLogger,
	}
}

func (u *UserCallBackBuilder) Build() func(db *gorm.DB) {
	return func(db *gorm.DB) {
		build(db, u.logger)
	}
}

func build(db *gorm.DB, logger *elog.Component) {
	ctx := db.Statement.Context
	appid, ok := appId(ctx)
	var tableName string
	var err error
	if ok {
		tableName, err = tableNameFromAppId(appid)
		if err != nil {
			logger.Error("获取别名失败", elog.FieldErr(err))
			return
		}
		db.Statement.Table = tableName
		return
	}
	appid, ok = appIdFromUserId(ctx)
	if ok {
		tableName, err = tableNameFromAppId(appid)
		if err != nil {
			logger.Error("获取别名失败", elog.FieldErr(err))
			return
		}
		db.Statement.Table = tableName
		return
	}
}

func appIdFromUserId(ctx context.Context) (uint, bool) {
	uid, ok := userId(ctx)
	if !ok {
		return 0, false
	}
	appid := snowflake.ID(uid).AppID()
	return appid, true
}

func userId(ctx context.Context) (int64, bool) {
	v := ctx.Value(uidCtxKey)
	if v == nil {
		return 0, false
	}
	uid, ok := v.(int64)
	return uid, ok
}

func appId(ctx context.Context) (uint, bool) {
	return ectx.GetAppIdFromCtx(ctx)
}

func tableNameFromAppId(appid uint) (string, error) {
	// 如果是0是webook不用加后缀
	if appid == 0 {
		return UserTableName, nil
	}
	appName, ok := appMap[appid]
	if !ok {
		return "", ErrUnknownAppid
	}
	return fmt.Sprintf("%s_%s", UserTableName, appName), nil
}
