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

package ioc

import (
	"context"

	"github.com/ecodeclub/webook/internal/payment/internal/service/wechat"
	"github.com/gotomicro/ego/core/econf"
	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/auth/verifiers"
	"github.com/wechatpay-apiv3/wechatpay-go/core/downloader"
	"github.com/wechatpay-apiv3/wechatpay-go/core/notify"
	"github.com/wechatpay-apiv3/wechatpay-go/core/option"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/native"
	"github.com/wechatpay-apiv3/wechatpay-go/utils"
)

func InitWechatClient(cfg WechatConfig) *core.Client {
	// 使用 utils 提供的函数从本地文件中加载商户私钥，商户私钥会用来生成请求的签名
	mchPrivateKey, err := utils.LoadPrivateKeyWithPath(
		// 注意这个文件我没有上传，所以你需要准备一个
		cfg.KeyPath,
	)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	// 使用商户私钥等初始化 client
	client, err := core.NewClient(
		ctx,
		option.WithWechatPayAutoAuthCipher(
			cfg.MchID, cfg.MchSerialNum,
			mchPrivateKey, cfg.MchKey),
	)
	if err != nil {
		panic(err)
	}
	return client
}

func InitWechatNativeService(native wechat.NativeAPIService, cfg WechatConfig) *wechat.NativePaymentService {
	return wechat.NewNativePaymentService(native, cfg.AppID, cfg.MchID, cfg.PaymentNotifyURL)
}

func InitNativeApiService(cli *core.Client) *native.NativeApiService {
	return &native.NativeApiService{
		Client: cli,
	}
}

func InitWechatNotifyHandler(cfg WechatConfig) *notify.Handler {
	certificateVisitor := downloader.MgrInstance().GetCertificateVisitor(cfg.MchID)
	// 3. 使用apiv3 key、证书访问器初始化 `notify.Handler`
	handler, err := notify.NewRSANotifyHandler(cfg.MchKey,
		verifiers.NewSHA256WithRSAVerifier(certificateVisitor))
	if err != nil {
		panic(err)
	}
	return handler
}

func InitWechatConfig() WechatConfig {
	return WechatConfig{
		AppID:            econf.GetString("wechat.appID"),
		MchID:            econf.GetString("wechat.mchID"),
		MchKey:           econf.GetString("wechat.mchKey"),
		MchSerialNum:     econf.GetString("wechat.mchSerialNum"),
		CertPath:         econf.GetString("wechat.certPath"),
		KeyPath:          econf.GetString("wechat.keyPath"),
		PaymentNotifyURL: econf.GetString("wechat.paymentNotifyURL"),
	}
}

type WechatConfig struct {
	AppID        string
	MchID        string
	MchKey       string
	MchSerialNum string

	// 证书
	CertPath string
	KeyPath  string

	PaymentNotifyURL string
}
