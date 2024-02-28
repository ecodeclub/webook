package testioc

import (
	"context"
	"database/sql"
	"time"

	"github.com/ecodeclub/ekit/retry"
	"github.com/ego-component/egorm"
	"github.com/gotomicro/ego/core/econf"
)

var db *egorm.Component

func InitDB() *egorm.Component {
	if db != nil {
		return db
	}
	econf.Set("mysql.user", map[string]string{"dsn": "root:root@tcp(localhost:13316)/webook"})
	WaitForDBSetup()
	db = egorm.Load("mysql.user").Build()
	return db
}

func WaitForDBSetup() {
	sqlDB, err := sql.Open("mysql", econf.GetStringMapString("mysql.user")["dsn"])
	if err != nil {
		panic(err)
	}
	const maxInterval = 10 * time.Second
	const maxRetries = 10
	strategy, err := retry.NewExponentialBackoffRetryStrategy(time.Second, maxInterval, maxRetries)
	if err != nil {
		panic(err)
	}

	const timeout = 5 * time.Second
	for {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		err = sqlDB.PingContext(ctx)
		cancel()
		if err == nil {
			break
		}
		next, ok := strategy.Next()
		if !ok {
			panic("WaitForDBSetup 重试失败......")
		}
		time.Sleep(next)
	}
}
