package testioc

import (
	"github.com/ego-component/egorm"
	"github.com/gotomicro/ego/core/econf"
)

var db *egorm.Component

func InitDB() *egorm.Component {
	if db != nil {
		return db
	}
	econf.Set("mysql.user", map[string]string{"dsn": "root:root@tcp(localhost:13316)/webook"})
	db = egorm.Load("mysql.user").Build()
	return db
}
