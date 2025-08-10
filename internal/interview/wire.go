// Copyright 2023 ecodeclub
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build wireinject

package interview

import (
	"sync"

	"github.com/gotomicro/ego/core/econf"

	"github.com/ecodeclub/webook/internal/email"
	"github.com/ecodeclub/webook/internal/email/aliyun"
	"github.com/ecodeclub/webook/internal/interview/internal/repository"
	"github.com/ecodeclub/webook/internal/interview/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/interview/internal/service"
	"github.com/ecodeclub/webook/internal/interview/internal/web"
	"github.com/ecodeclub/webook/internal/pkg/pdf"
	"github.com/ego-component/egorm"
	"github.com/google/wire"
	"gorm.io/gorm"
)

type (
	JourneyHandler = web.InterviewJourneyHandler
	OfferHandler   = web.OfferHandler
)

func InitModule(db *egorm.Component) (*Module, error) {
	wire.Build(
		initDAO,
		repository.NewInterviewRepository,
		service.NewInterviewService,
		web.NewInterviewJourneyHandler,
		initOfferHdl,
		wire.Struct(new(Module), "*"),
	)
	return nil, nil
}

var initOnce sync.Once

func initDAO(db *gorm.DB) dao.InterviewDAO {
	initOnce.Do(func() {
		err := dao.InitTables(db)
		if err != nil {
			panic(err)
		}
	})
	return dao.NewGORMInterviewDAO(db)
}

func initOfferHdl() *web.OfferHandler {
	emailCli := initEmailClient()
	converter := initPDFConverter()
	tmpl := initOfferTemplate()
	oSvc := service.NewOfferService(emailCli, converter, tmpl)
	return web.NewOfferHandler(oSvc)
}

func initOfferTemplate() string {
	// 从配置中读取模板
	return econf.GetString("offer.template")
}

func initPDFConverter() pdf.Converter {
	type cfg struct {
		Endpoint string `yaml:"endpoint"`
	}
	var c cfg
	// 读取 pdf 服务地址，例如: pdf.endpoint: http://localhost:9999/pdf/convert
	err := econf.UnmarshalKey("pdf", &c)
	if err != nil {
		panic(err)
	}
	return pdf.NewRemotePDFConverter(c.Endpoint)
}

func initEmailClient() email.Service {
	type Cfg struct {
		AccessID     string `yaml:"accessId"`
		AccessSecret string `yaml:"accessSecret"`
		AccountName  string `yaml:"accountName"`
	}
	var cfg Cfg
	// email.ali 配置
	_ = econf.UnmarshalKey("email.ali", &cfg)
	cli, err := aliyun.NewAliyunDirectMailAPI(cfg.AccessID, cfg.AccessSecret, cfg.AccountName)
	if err != nil {
		panic(err)
	}
	return cli
}
