package ioc

import (
	"github.com/ecodeclub/webook/internal/sms/client"
	"github.com/gotomicro/ego/core/econf"
)

func initAliSMSClient() client.Client {
	type Config struct {
		Mock      bool   `json:"mock"`
		SecretID  string `yaml:"secretID"`
		SecretKey string `yaml:"secretKey"`
	}
	var cfg Config
	err := econf.UnmarshalKey("sms.aliyun", &cfg)
	if err != nil {
		panic(err)
	}
	if cfg.Mock {
		return client.NewConsoleClient()
	}
	aliClient, err := client.NewAliyunSMS(cfg.SecretID, cfg.SecretKey)
	if err != nil {
		panic(err)
	}
	return aliClient
}
