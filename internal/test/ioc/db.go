package testioc

import (
	"bytes"
	"os"
	"path/filepath"

	"github.com/ecodeclub/webook/ioc"
	"github.com/ego-component/egorm"
	"github.com/gotomicro/ego/core/econf"
	"gopkg.in/yaml.v3"
)

var db *egorm.Component

func InitDB() *egorm.Component {
	if db != nil {
		return db
	}
	if err := loadConfig(); err != nil {
		panic(err)
	}
	ioc.WaitForDBSetup(econf.GetStringMapString("mysql")["dsn"])
	db = egorm.Load("mysql").Build()
	return db
}

func loadConfig() error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	path := filepath.Clean(dir + "../../../../../config/local.yaml")
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return econf.LoadFromReader(bytes.NewReader(content), yaml.Unmarshal)
}
