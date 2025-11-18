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
  "io"

  chatv1 "github.com/ecodeclub/ai-gateway-go/api/proto/gen/chat/v1"
  "github.com/ecodeclub/ekit/slice"
  "github.com/ecodeclub/ginx"
  "github.com/ecodeclub/ginx/session"
  "github.com/gin-gonic/gin"
  "github.com/gotomicro/ego/core/elog"
)

type ChatHandler struct {
	client chatv1.ServiceClient
	logger *elog.Component
}

func NewChatHandler(client chatv1.ServiceClient) *ChatHandler {
	return &ChatHandler{
		client: client,
		logger: elog.DefaultLogger,
	}
}

func (h *ChatHandler) MemberRoutes(server *gin.Engine) {
	group := server.Group("/chat")
	group.POST("/save", ginx.BS[SaveChatRequest](h.Save))
	group.POST("/detail", ginx.BS[ChatSN](h.Detail))
	group.POST("/list", ginx.BS[Page](h.List))
	// 用户有一个新的输入
	group.POST("/stream", ginx.BS[StreamRequest](h.Stream))
}

func (h *ChatHandler) Stream(ctx *ginx.Context, req StreamRequest, sess session.Session) (ginx.Result, error) {
	// 先把 ID 拿到
	ch := ctx.EventStreamResp()
	uid := sess.Claims().Uid
	resp, err := h.client.Stream(ctx, &chatv1.StreamRequest{
		ChatSn:             req.SN,
		Uid:                uid,
		Input: &chatv1.UserInput{
			Content: req.Msg.Content,
		},
	})

	if err != nil {
		ch <- h.buildMsg(ErrEvt, "系统错误")
		return systemErrorResult, ginx.ErrNoResponse
	}

	ctx.Stream(func(w io.Writer) bool {
		select {
		case <-ctx.Done():
			return false
		default:
			event := &chatv1.StreamResponse{}
			err = resp.RecvMsg(event)

			// 正常结束
			if errors.Is(err, io.EOF) {
				h.logger.Debug("正常结束---------------")
				_, _ = w.Write(h.buildMsg(EndEvt, ""))
				return false
			}

			if err != nil {
				h.logger.Error("接收流式数据失败", elog.FieldErr(err))
				_, _ = w.Write(h.buildMsg(ErrEvt, "系统错误"))
				return false
			}

			switch val := event.Event.(type) {
			case *chatv1.StreamResponse_Error:
				_, _ = w.Write(h.buildMsg(ErrEvt, ""))
				return true
			case *chatv1.StreamResponse_Delta:
				_, _ = w.Write(h.buildMsg(MsgEvt, val.Delta.Content))
				return true
			case *chatv1.StreamResponse_StepUpdate:
				_, _ = w.Write(h.buildMsg(StepUpdateEvt, val.StepUpdate.Name))
				return true
			default:
				h.logger.Error("未知事件类型", elog.Any("event", val))
				return true
			}
		}
	})
	return ginx.Result{}, ginx.ErrNoResponse
}

func (h *ChatHandler) buildMsg(typ string, content string) []byte {
	val, _ := json.Marshal(Event{
		Type: typ,
		Data: EvtMsg{
			Content: content,
		},
	})
	val = append(val, '\n')
	h.logger.Debug("", elog.String("event", typ), elog.String("content", content))
	return val
}

func (h *ChatHandler) Detail(ctx *ginx.Context, req ChatSN, sess session.Session) (ginx.Result, error) {
	resp, err := h.client.Detail(ctx, &chatv1.DetailRequest{
		Sn: req.SN,
	})
	if err != nil {
		return systemErrorResult, err
	}
	chat := resp.GetChat()
	return ginx.Result{
		Data: newChat(chat),
	}, nil
}

func (h *ChatHandler) List(ctx *ginx.Context, req Page, sess session.Session) (ginx.Result, error) {
	resp, err := h.client.List(ctx, &chatv1.ListRequest{
		Uid:    sess.Claims().Uid,
		Offset: req.Offset,
		Limit:  req.Limit,
	})
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: slice.Map(resp.GetChats(), func(idx int, src *chatv1.Chat) Chat {
			return newChat(src)
		}),
	}, nil
}

func (h *ChatHandler) Save(ctx *ginx.Context, req SaveChatRequest, sess session.Session) (ginx.Result, error) {
	resp, err := h.client.Save(ctx, &chatv1.SaveRequest{Chat: &chatv1.Chat{
		Uid:   sess.Claims().Uid,
		Sn:    req.SN,
		Title: req.Title,
	}})
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: resp.GetSn(),
	}, nil
}
