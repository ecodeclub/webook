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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	chatv1 "github.com/ecodeclub/webook/api/proto/gen/chat/v1"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/econf"
	"github.com/gotomicro/ego/core/elog"
	sts "github.com/tencentyun/qcloud-cos-sts-sdk/go"
)

var _ ginx.Handler = &MockInterviewHandler{}

type MockInterviewHandler struct {
	client chatv1.ServiceClient
	logger *elog.Component
}

func NewMockInterviewHandler(client chatv1.ServiceClient) *MockInterviewHandler {
	return &MockInterviewHandler{
		client: client,
		logger: elog.DefaultLogger.With(elog.FieldComponent("MockInterviewHandler")),
	}
}

func (h *MockInterviewHandler) PublicRoutes(server *gin.Engine) {
	//TODO implement me
	panic("implement me")
}

func (h *MockInterviewHandler) PrivateRoutes(server *gin.Engine) {
	server.POST("/ai/mock_interview/create", ginx.BS(h.CreateMockInterview))
	server.POST("/ai/mock_interview/stream", h.Stream)
	// 注意：这个API是为了方便本地测试，正式前端页面不应该请求这个API，应该用正确的方法获取cos临时凭证
	server.GET("/ai/mock_interview/cos/temp-credentials", h.GetCOSTempCredentials)
}

func (h *MockInterviewHandler) CreateMockInterview(ctx *ginx.Context, req CreateMockInterviewReq, sess session.Session) (ginx.Result, error) {
	h.logger.Debug("创建会话")
	// 根据uid 和 title 拼接请求，调用GRPC来请求
	resp, err := h.client.Save(ctx.Request.Context(), &chatv1.SaveRequest{
		Chat: &chatv1.Chat{
			Uid:   sess.Claims().Uid,
			Title: req.Title,
		},
	})
	if err != nil {
		return systemErrorResult, fmt.Errorf("%w: 创建模拟面试失败", err)
	}
	return ginx.Result{Msg: "ok", Data: resp.Sn}, nil
}

func (h *MockInterviewHandler) Stream(ctx *gin.Context) {
	// 1. 获取 session
	gtx := &ginx.Context{Context: ctx}
	sess, err := session.Get(gtx)
	if err != nil {
		h.logger.Error("获取 Session 失败", elog.FieldErr(err))
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	uid := sess.Claims().Uid

	// 2. 解析请求参数
	var req StreamMockInterviewReq
	if err := ctx.Bind(&req); err != nil {
		h.logger.Error("绑定参数失败", elog.FieldErr(err))
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}

	// 3. 构建 gRPC 请求
	stream, err := h.client.StreamV1(gtx.Request.Context(), &chatv1.StreamV1Request{
		ChatSn: req.InterviewID,
		Input: &chatv1.UserInput{
			Content:  req.Content,
			AudioUrl: req.AudioURL,
		},
		InvocationConfigId: req.ConfigID,
		Uid:                uid,
		Key:                "", // 未使用，传空字符串
	})
	if err != nil {
		h.logger.Error("调用 StreamV1 失败", elog.FieldErr(err))
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	// 4. 设置 SSE 响应头
	ctx.Writer.Header().Set("Content-Type", "text/event-stream")
	ctx.Writer.Header().Set("Cache-Control", "no-cache")
	ctx.Writer.Header().Set("Connection", "keep-alive")

	flusher, ok := ctx.Writer.(http.Flusher)
	if !ok {
		h.logger.Error("不支持流式响应")
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	// 5. 转发流式响应
	deltaCount := 0
	for {

		resp, err := stream.Recv()

		if errors.Is(err, io.EOF) {
			_, err := fmt.Fprintf(ctx.Writer, "event: done\ndata: {}\n\n")
			if err != nil {
				h.logger.Error("发送done事件失败")
				ctx.AbortWithStatus(http.StatusInternalServerError)
				return
			}
			flusher.Flush()
			h.logger.Debug("Stream 完成", elog.Int("DeltaCount", deltaCount))
			break
		}

		if err != nil {
			h.logger.Error("Stream 错误", elog.FieldErr(err))
			_, err = fmt.Fprintf(ctx.Writer, "event: error\ndata: {\"message\": \"%s\"}\n\n", err.Error())
			if err != nil {
				h.logger.Error("发送error事件失败")
				ctx.AbortWithStatus(http.StatusInternalServerError)
				return
			}
			flusher.Flush()
			break
		}

		// 转换为 JSON 并发送
		data, _ := json.Marshal(resp)
		_, err = fmt.Fprintf(ctx.Writer, "data: %s\n\n", data)
		if err != nil {
			h.logger.Error("发送data事件失败")
			ctx.AbortWithStatus(http.StatusInternalServerError)
			break
		}
		flusher.Flush()

		// 日志（Delta 事件）
		if d := resp.GetDelta(); d != nil {
			deltaCount++
			h.logger.Debug("Delta 事件", elog.Int("Count", deltaCount), elog.String("Content", d.Content))
		}
	}
}

// GetCOSTempCredentials 获取腾讯云 COS 临时密钥
func (h *MockInterviewHandler) GetCOSTempCredentials(ctx *gin.Context) {
	h.logger.Debug("请求 COS 临时密钥")

	// 1. 读取 COS 配置
	type COSConfig struct {
		SecretID  string `yaml:"secretID"`
		SecretKey string `yaml:"secretKey"`
		Bucket    string `yaml:"bucket"`
		Region    string `yaml:"region"`
	}
	var cfg COSConfig
	if err := econf.UnmarshalKey("cos", &cfg); err != nil {
		h.logger.Error("读取 COS 配置失败", elog.FieldErr(err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "获取临时密钥失败: 配置错误"})
		return
	}

	// 验证配置
	if cfg.SecretID == "" || cfg.SecretKey == "" || cfg.Bucket == "" || cfg.Region == "" {
		h.logger.Error("COS 配置不完整")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "COS 配置不完整"})
		return
	}

	// 2. 创建 STS 客户端
	stsClient := sts.NewClient(cfg.SecretID, cfg.SecretKey, nil)

	// 3. 配置临时密钥选项
	// 从 bucket 名称中提取 appid（格式：bucketname-appid，如 webook-1314583317）
	parts := strings.Split(cfg.Bucket, "-")
	appid := parts[len(parts)-1]

	opt := &sts.CredentialOptions{
		DurationSeconds: 1800, // 30分钟
		Region:          cfg.Region,
		Policy: &sts.CredentialPolicy{
			Statement: []sts.CredentialPolicyStatement{
				{
					Action: []string{
						"cos:PutObject",
						"cos:PostObject",
					},
					Effect: "allow",
					// 标准格式（包含 uid/{appid}）
					Resource: []string{
						fmt.Sprintf("qcs::cos:%s:uid/%s:%s/audio-temp/*", cfg.Region, appid, cfg.Bucket),
					},
				},
			},
		},
	}

	// 4. 获取临时密钥
	credential, err := stsClient.GetCredential(opt)
	if err != nil {
		h.logger.Error("获取临时密钥失败", elog.FieldErr(err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "获取临时密钥失败"})
		return
	}

	// 5. 返回给前端
	response := COSTempCredentialsResp{
		TmpSecretId:  credential.Credentials.TmpSecretID,
		TmpSecretKey: credential.Credentials.TmpSecretKey,
		SessionToken: credential.Credentials.SessionToken,
		StartTime:    int64(credential.StartTime),
		ExpiredTime:  int64(credential.ExpiredTime),
		Bucket:       cfg.Bucket,
		Region:       cfg.Region,
	}

	h.logger.Debug("临时密钥已生成", elog.Int("expiredTime", credential.ExpiredTime))
	ctx.JSON(http.StatusOK, response)
}
