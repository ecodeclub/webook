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

package client

import (
	"github.com/ecodeclub/ekit/slice"
	"github.com/gotomicro/ego/core/elog"
	"github.com/lithammer/shortuuid/v4"
)

type ConsoleClient struct {
	logger *elog.Component
}

func NewConsoleClient() *ConsoleClient {
	return &ConsoleClient{
		logger: elog.DefaultLogger,
	}
}

func (c *ConsoleClient) Send(req SendReq) (SendResp, error) {
	reqID := shortuuid.New()
	c.logger.Error("发送短信:", elog.Any("req", req))
	return SendResp{
		RequestID: reqID,
		PhoneNumbers: slice.ToMapV(req.PhoneNumbers, func(element string) (string, SendRespStatus) {
			return element, SendRespStatus{
				Code: "OK",
			}
		}),
	}, nil
}
