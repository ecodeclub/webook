package testioc

import (
	"github.com/ecodeclub/webook/ioc"
	"github.com/ego-component/egorm"
	"github.com/gotomicro/ego/core/econf"
)

var db *egorm.Component

func InitDB() *egorm.Component {
	if db != nil {
		return db
	}
	econf.Set("mysql", map[string]any{
		"dsn":   "webook:webook@tcp(localhost:13316)/webook?charset=utf8mb4&collation=utf8mb4_general_ci&parseTime=True&loc=Local&timeout=1s&readTimeout=3s&writeTimeout=3s",
		"debug": false,
	})
	ioc.WaitForDBSetup(econf.GetStringMapString("mysql")["dsn"])
	db = egorm.Load("mysql").Build()
	return db
}
