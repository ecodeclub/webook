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
  "github.com/ecodeclub/ginx"
  "github.com/gotomicro/ego/core/elog"
  "google.golang.org/grpc"
)

type BaseHandler struct {
  logger *elog.Component
}

func(h *BaseHandler) stream(ctx *ginx.Context, resp grpc.ServerStreamingClient[chatv1.StreamResponse]) {
  ctx.Stream(func(w io.Writer) bool {
    select {
    case <-ctx.Done():
      return false
    default:
      event := &chatv1.StreamResponse{}
      err := resp.RecvMsg(event)
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
}

func (h *BaseHandler) buildMsg(typ string, content string) []byte {
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
