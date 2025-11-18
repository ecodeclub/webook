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
	chatv1 "github.com/ecodeclub/ai-gateway-go/api/proto/gen/chat/v1"
	"github.com/ecodeclub/ekit/slice"
)

type StreamRequest struct {
	SN  string  `json:"sn"`
	Msg Message `json:"msg"`
}

type SaveChatRequest struct {
	SN    string `json:"sn"`
	Title string `json:"title"`
  // 描述是什么业务场景
  BizID int64  `json:"bizID"`
}

type ChatSN struct {
	SN string `json:"sn"`
}

type Page struct {
	Offset int64 `json:"offset"`
	Limit  int64 `json:"limit"`
}

type Chat struct {
	ID    int64     `json:"id"`
	SN    string    `json:"sn"`
	Title string    `json:"title"`
	Msgs  []Message `json:"msgs"`
	Ctime int64     `json:"ctime"`
}

func newChat(chat *chatv1.Chat) Chat {
	return Chat{
		SN:    chat.Sn,
		Title: chat.Title,
		Msgs: slice.Map(chat.Msgs, func(idx int, src *chatv1.Message) Message {
			return newMessage(src)
		}),
		Ctime: chat.Ctime,
	}
}

type Message struct {
	ID      int64  `json:"id"`
	Role    string `json:"role"`
	Content string `json:"content"`
}

func newMessage(msg *chatv1.Message) Message {
	return Message{
		ID:      msg.Id,
		Content: msg.Content,
		Role:    msg.Role,
	}
}
