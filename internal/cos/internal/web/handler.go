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

package web

import (
	"fmt"
	"net/http"
	"time"

	"github.com/ecodeclub/ginx"
	"github.com/gin-gonic/gin"
	sts "github.com/tencentyun/qcloud-cos-sts-sdk/go"
)

var _ ginx.Handler = &Handler{}

type Handler struct {
	client *sts.Client
	// 临时密钥的权限
	actions []string

	appID  string
	bucket string
	region string
}

func NewHandler(secretID, secretKey, appid, bucket,
	region string) *Handler {
	c := sts.NewClient(
		secretID,
		secretKey,
		http.DefaultClient,
	)
	return &Handler{client: c,
		region: region,
		appID:  appid,
		bucket: bucket,
		actions: []string{
			// 简单上传
			"name/cos:PostObject",
			"name/cos:PutObject",
			// 分片上传
			"name/cos:InitiateMultipartUpload",
			"name/cos:ListMultipartUploads",
			"name/cos:ListParts",
			"name/cos:UploadPart",
			"name/cos:CompleteMultipartUpload",
		},
	}
}

func (h *Handler) PrivateRoutes(server *gin.Engine) {
	cos := server.Group("/cos")
	cos.POST("/authorization", ginx.B(h.TempAuthCode))
}

func (h *Handler) PublicRoutes(server *gin.Engine) {
}

func (h *Handler) TempAuthCode(ctx *ginx.Context, req TmpAuthCodeReq) (ginx.Result, error) {
	// 策略概述 https://cloud.tencent.com/document/product/436/18023
	// 这里改成允许的路径前缀，可以根据自己网站的用户登录态判断允许上传的具体路径，例子： a.jpg 或者 a/* 或者 * (使用通配符*存在重大安全风险, 请谨慎评估使用)
	// 存储桶的命名格式为 BucketName-APPID，此处填写的 bucket 必须为此格式
	resource := fmt.Sprintf("qcs::cos:%s:uid/%s:%s-%s/%s",
		h.region, h.appID,
		h.bucket, h.appID, req.Key)
	opt := &sts.CredentialOptions{
		DurationSeconds: int64(time.Hour.Seconds()),
		Region:          h.region,
		Policy: &sts.CredentialPolicy{
			Statement: []sts.CredentialPolicyStatement{
				{
					Action: h.actions,
					Effect: "allow",
					Resource: []string{
						resource,
					},
					Condition: map[string]map[string]interface{}{
						"string_equal": {
							"cos:content-type": req.Type,
						},
					},
				},
			},
		},
	}

	// case 1 请求临时密钥
	res, err := h.client.GetCredential(opt)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: COSTmpAuthCode{
			SecretId:     res.Credentials.TmpSecretID,
			SecretKey:    res.Credentials.TmpSecretKey,
			SessionToken: res.Credentials.SessionToken,
			StartTime:    res.StartTime,
			ExpiredTime:  res.ExpiredTime,
		},
	}, nil
}
