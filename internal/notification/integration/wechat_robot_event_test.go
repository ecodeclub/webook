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

//go:build e2e

package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/notification/event"
	"github.com/ecodeclub/webook/internal/notification/wechat/consumer"
	"github.com/ecodeclub/webook/internal/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestWechatRobotEvent(t *testing.T) {
	// todo 暂时跳过, 后续查看原因
	t.Skip()
	suite.Run(t, new(WechatRobotEventTestSuite))
}

type WechatRobotEventTestSuite struct {
	suite.Suite
}

func (s *WechatRobotEventTestSuite) TestNew() {
	t := s.T()

	testCases := []struct {
		name            string
		before          func(t *testing.T, ctrl *gomock.Controller) mq.MQ
		cfg             *consumer.WechatRobotConfig
		errorAssertFunc assert.ErrorAssertionFunc
	}{
		{
			name: "创建成功",
			before: func(t *testing.T, ctrl *gomock.Controller) mq.MQ {
				t.Helper()
				mockMQ := mocks.NewMockMQ(ctrl)
				mockConsumer := mocks.NewMockConsumer(ctrl)
				mockMQ.EXPECT().Consumer(gomock.Any(), gomock.Any()).Return(mockConsumer, nil).Times(1)
				return mockMQ
			},
			cfg: &consumer.WechatRobotConfig{ChatRobots: map[string]string{
				"adminRobot": "http://localhost:9293/hello",
			}},
			errorAssertFunc: assert.NoError,
		},
		{
			name: "创建失败",
			before: func(t *testing.T, ctrl *gomock.Controller) mq.MQ {
				t.Helper()
				mockMQ := mocks.NewMockMQ(ctrl)
				mockMQ.EXPECT().Consumer(gomock.Any(), gomock.Any()).Return(nil, errors.New("fake error")).Times(1)
				return mockMQ
			},
			cfg: &consumer.WechatRobotConfig{ChatRobots: map[string]string{
				"adminRobot": "http://localhost:9293/hello",
			}},
			errorAssertFunc: assert.Error,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockMQ := tc.before(t, ctrl)
			_, err := consumer.NewWechatRobotEventConsumer(mockMQ, tc.cfg)
			tc.errorAssertFunc(t, err)
		})
	}
}
func (s *WechatRobotEventTestSuite) TestStart() {
	t := s.T()

	testCases := []struct {
		name    string
		before  func(t *testing.T, ctrl *gomock.Controller) mq.MQ
		ctxFunc func(t *testing.T) context.Context
		cfg     *consumer.WechatRobotConfig
	}{
		{
			name: "启动正常",
			before: func(t *testing.T, ctrl *gomock.Controller) mq.MQ {
				t.Helper()
				mockMQ := mocks.NewMockMQ(ctrl)
				mockConsumer := mocks.NewMockConsumer(ctrl)
				mockMQ.EXPECT().Consumer(gomock.Any(), gomock.Any()).Return(mockConsumer, nil).Times(1)
				mockConsumer.EXPECT().Consume(gomock.Any()).Return(nil, context.Canceled).Times(1)
				return mockMQ
			},
			ctxFunc: func(t *testing.T) context.Context {
				t.Helper()
				ctx, cancel := context.WithCancel(t.Context())
				cancel()
				return ctx
			},
			cfg: &consumer.WechatRobotConfig{ChatRobots: map[string]string{
				"adminRobot": "http://localhost:9293/hello",
			}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockMQ := tc.before(t, ctrl)
			c, err := consumer.NewWechatRobotEventConsumer(mockMQ, tc.cfg)
			require.NoError(t, err)
			c.Start(tc.ctxFunc(t))
			time.Sleep(3 * time.Second)
		})
	}
}

func (s *WechatRobotEventTestSuite) TestConsume() {
	t := s.T()

	mockHTTPServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Hello, adminRobot!")
	}))

	defer mockHTTPServer.Close()

	testCases := []struct {
		name            string
		before          func(t *testing.T, ctrl *gomock.Controller) mq.MQ
		cfg             *consumer.WechatRobotConfig
		errorAssertFunc assert.ErrorAssertionFunc
	}{
		{
			name: "发送请求成功",
			before: func(t *testing.T, ctrl *gomock.Controller) mq.MQ {
				t.Helper()
				mockMQ := mocks.NewMockMQ(ctrl)
				mockConsumer := mocks.NewMockConsumer(ctrl)

				mockMQ.EXPECT().Consumer(gomock.Any(), gomock.Any()).Return(mockConsumer, nil).Times(1)

				evt := &event.WechatRobotEvent{
					Robot:      "adminRobot",
					RawContent: "Hello, adminRobot!",
				}

				msg, err := json.Marshal(evt)
				require.NoError(t, err)

				mockConsumer.EXPECT().Consume(gomock.Any()).Return(&mq.Message{Value: msg}, nil).Times(1)
				return mockMQ
			},
			cfg: &consumer.WechatRobotConfig{ChatRobots: map[string]string{
				"adminRobot": mockHTTPServer.URL,
			}},
			errorAssertFunc: assert.NoError,
		},
		{
			name: "发送请求失败_从MQ获取消息失败",
			before: func(t *testing.T, ctrl *gomock.Controller) mq.MQ {
				t.Helper()
				mockMQ := mocks.NewMockMQ(ctrl)
				mockConsumer := mocks.NewMockConsumer(ctrl)
				mockMQ.EXPECT().Consumer(gomock.Any(), gomock.Any()).Return(mockConsumer, nil).Times(1)
				mockConsumer.EXPECT().Consume(gomock.Any()).Return(nil, errors.New("fake error")).Times(1)
				return mockMQ
			},
			cfg: &consumer.WechatRobotConfig{ChatRobots: map[string]string{
				"adminRobot": mockHTTPServer.URL,
			}},
			errorAssertFunc: assert.Error,
		},
		{
			name: "发送请求失败_消息体非法",
			before: func(t *testing.T, ctrl *gomock.Controller) mq.MQ {
				t.Helper()
				mockMQ := mocks.NewMockMQ(ctrl)
				mockConsumer := mocks.NewMockConsumer(ctrl)
				mockMQ.EXPECT().Consumer(gomock.Any(), gomock.Any()).Return(mockConsumer, nil).Times(1)
				mockConsumer.EXPECT().Consume(gomock.Any()).Return(&mq.Message{Value: []byte("invalid msg")}, nil).Times(1)
				return mockMQ
			},
			cfg: &consumer.WechatRobotConfig{ChatRobots: map[string]string{
				"adminRobot": mockHTTPServer.URL,
			}},
			errorAssertFunc: assert.Error,
		},
		{
			name: "发送请求失败_消息体值非法",
			before: func(t *testing.T, ctrl *gomock.Controller) mq.MQ {
				t.Helper()
				mockMQ := mocks.NewMockMQ(ctrl)
				mockConsumer := mocks.NewMockConsumer(ctrl)

				mockMQ.EXPECT().Consumer(gomock.Any(), gomock.Any()).Return(mockConsumer, nil).Times(1)

				evt := &event.WechatRobotEvent{
					Robot:      "adminRobots",
					RawContent: "Hello, adminRobot!",
				}

				msg, err := json.Marshal(evt)
				require.NoError(t, err)

				mockConsumer.EXPECT().Consume(gomock.Any()).Return(&mq.Message{Value: msg}, nil).Times(1)
				return mockMQ
			},
			cfg: &consumer.WechatRobotConfig{ChatRobots: map[string]string{
				"adminRobot": mockHTTPServer.URL,
			}},
			errorAssertFunc: assert.Error,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockMQ := tc.before(t, ctrl)
			eventConsumer, err := consumer.NewWechatRobotEventConsumer(mockMQ, tc.cfg)
			require.NoError(t, err)
			tc.errorAssertFunc(t, eventConsumer.Consume(t.Context()))
		})
	}

}
