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

  chatv1 "github.com/ecodeclub/ai-gateway-go/api/proto/gen/chat/v1"
  "github.com/ecodeclub/ginx"
  "github.com/ecodeclub/ginx/session"
  "github.com/ecodeclub/webook/internal/ai/internal/service"
  "github.com/gin-gonic/gin"
  "github.com/gotomicro/ego/core/elog"
  "github.com/lithammer/shortuuid/v4"
)

var _ ginx.Handler = &MockInterviewHandler{}

type MockInterviewHandler struct {
  BaseHandler
	client chatv1.ServiceClient
	svc    service.MockInterviewService
	logger *elog.Component
}


func NewMockInterviewHandler(client chatv1.ServiceClient, svc service.MockInterviewService) *MockInterviewHandler {
	return &MockInterviewHandler{
		client: client,
		svc:    svc,
		BaseHandler: BaseHandler{
      logger: elog.DefaultLogger.With(elog.FieldComponent("MockInterviewHandler")),
    },
	}
}

func (h *MockInterviewHandler) PublicRoutes(server *gin.Engine) {
	//TODO implement me
	panic("implement me")
}

func (h *MockInterviewHandler) PrivateRoutes(server *gin.Engine) {
	// 注意：这个API是为了方便本地测试，正式前端页面不应该请求这个API，应该用正确的方法获取cos临时凭证
	server.POST("/ai/mock_interview/start", ginx.BS[StartMockInterviewRequest](h.Start))
	server.POST("/ai/mock_interview/next", ginx.BS[NextQuestionRequest](h.NextQuestion))
}

// Start 后续处理 title 的拼接问题
func (h *MockInterviewHandler) Start(ctx *ginx.Context, req StartMockInterviewRequest, sess session.Session) (ginx.Result, error) {
  const bizID = 2
  resp, err := h.client.Save(ctx, &chatv1.SaveRequest{
    BizId: bizID,
    Chat: &chatv1.Chat{
      Uid:   sess.Claims().Uid,
      Title: "模拟面试",
    },
  })
  if err != nil {
    return systemErrorResult, err
  }

  return ginx.Result{
    Data: resp.GetSn(),
  }, nil
}

func (h *MockInterviewHandler) NextQuestion(ctx *ginx.Context, req NextQuestionRequest, sess session.Session) (ginx.Result, error) {
  resp, err := h.client.Stream(ctx, &chatv1.StreamRequest{
    ChatSn: req.SN,
    // 上一题的回答
    Input: &chatv1.UserInput{
      Content: req.Answer,
    },
    Uid:   sess.Claims().Uid,
    // 这个其实现在暂时用不上，后续需要前端来传递，避免重复请求
    Key: shortuuid.New(),
    State: req.State,
  })
  if err != nil {
    return systemErrorResult, fmt.Errorf("调用 AI stream 接口失败 %w", err)
  }
  h.stream(ctx, resp)
  return ginx.Result{}, ginx.ErrNoResponse
}
