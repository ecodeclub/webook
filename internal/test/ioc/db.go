package testioc

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/ego-component/egorm"
	"github.com/gotomicro/ego/core/econf"
)

var db *egorm.Component

func InitDB() *egorm.Component {
	if db != nil {
		return db
	}
	sqlDB, err := sql.Open("mysql", "root:root@tcp(localhost:13316)/mysql")
	if err != nil {
		panic(err)
	}
	for i := 0; i < 300; i++ {
		log.Println("等待测试 DB 启动")
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
		err = sqlDB.PingContext(ctx)
		cancel()
		if err == nil {
			break
		}
	}
	econf.Set("mysql.user", map[string]string{"dsn": "root:root@tcp(localhost:13316)/mysql"})
	db = egorm.Load("mysql.user").Build()
	return db
}
