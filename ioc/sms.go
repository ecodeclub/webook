package ioc

import (
	"github.com/ecodeclub/webook/internal/sms/client"
	"github.com/gotomicro/ego/core/econf"
)

func initTencentCloudSMS() client.Client {
	type Config struct {
		SecretID  string `yaml:"secretID"`
		SecretKey string `yaml:"secretKey"`
	}
	var cfg Config
	err := econf.UnmarshalKey("sms.aliyun", &cfg)
	if err != nil {
		panic(err)
	}
	aliClient, err := client.NewAliyunSMS(cfg.SecretID, cfg.SecretKey)
	if err != nil {
		panic(err)
	}
	return aliClient
}
