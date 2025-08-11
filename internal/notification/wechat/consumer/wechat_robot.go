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

package consumer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/notification/event"
	"github.com/gotomicro/ego/core/elog"
)

type Text struct {
	Content string `json:"content"`
}
type WechatRobotMessage struct {
	MsgType string `json:"msgtype"`
	Text    Text   `json:"text"`
}

type HTTPPOSTFunc func(url, contentType string, body io.Reader) (resp *http.Response, err error)

type WechatRobotConfig struct {
	ChatRobots map[string]string `yaml:"chatRobots"`
}

type WechatRobotEventConsumer struct {
	consumer mq.Consumer
	config   *WechatRobotConfig
	logger   *elog.Component
}

func NewWechatRobotEventConsumer(q mq.MQ, config *WechatRobotConfig) (*WechatRobotEventConsumer, error) {
	groupID := "notification.wechat"
	consumer, err := q.Consumer(event.WechatRobotEventName, groupID)
	if err != nil {
		return nil, err
	}
	return &WechatRobotEventConsumer{
		consumer: consumer,
		config:   config,
		logger:   elog.DefaultLogger.With(elog.FieldComponent("notification.wechat.consumer")),
	}, nil
}

// Start 后面要考虑借助 ctx 来优雅退出
func (c *WechatRobotEventConsumer) Start(ctx context.Context) {
	go func() {
		for {
			err := c.Consume(ctx)
			if err != nil {
				log.Printf("err = %#v\n", err)
				c.logger.Error("消费微信机器人事件失败", elog.FieldErr(err))
			}
			if ctx.Err() != nil {
				return
			}
		}
	}()
}

func (c *WechatRobotEventConsumer) Consume(ctx context.Context) error {
	msg, err := c.consumer.Consume(ctx)
	if err != nil {
		return fmt.Errorf("获取消息失败: %w", err)
	}
	var evt event.WechatRobotEvent
	err = json.Unmarshal(msg.Value, &evt)
	if err != nil {
		return fmt.Errorf("解析消息失败: %w", err)
	}
	// 获取机器人地址
	webhookURL, ok := c.config.ChatRobots[evt.Robot]
	if !ok {
		c.logger.Error("未知Robot消息", elog.Any("event", evt))
		return errors.New("未知Robot消息")
	}
	// 构造消息体
	data, err := json.Marshal(&WechatRobotMessage{MsgType: "text", Text: Text{Content: evt.RawContent}})
	if err != nil {
		return fmt.Errorf("序列化微信Robot消息失败: %w", err)
	}
	// 发送消息
	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("向微信发送请求失败: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("微信处理请求失败: %s", http.StatusText(resp.StatusCode))
	}
	return nil
}
