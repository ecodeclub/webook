package ioc

import (
	"github.com/ecodeclub/webook/internal/sms/client"
	"github.com/gotomicro/ego/core/econf"
)

func initTencentCloudSMS() client.Client {
	type Config struct {
		RegionID  string `yaml:"regionID"`
		SecretID  string `yaml:"secretID"`
		SecretKey string `yaml:"secretKey"`
		AppID     string `yaml:"appID"`
	}
	var cfg Config
	err := econf.UnmarshalKey("sms.tencentcloud", &cfg)
	if err != nil {
		panic(err)
	}
	tencentClient, err := client.NewTencentCloudSMS(cfg.RegionID, cfg.SecretID, cfg.SecretKey, cfg.AppID)
	if err != nil {
		panic(err)
	}
	return tencentClient
}
