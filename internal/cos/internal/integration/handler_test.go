//go:build e2e

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
package integration

import (
	"os"
	"testing"

	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/webook/internal/cos/internal/web"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// HandlerTestSuite 在本地无法通过这个测试是正常的，可以忽略。
type HandlerTestSuite struct {
	suite.Suite
	handler *web.Handler
}

func (s *HandlerTestSuite) SetupSuite() {
	appID := os.Getenv("COS_APP_ID")
	if appID == "" {
		s.FailNow("未找到 appID，本地启动可以忽略")
	}
	bucket := os.Getenv("COS_BUCKET")
	if bucket == "" {
		s.FailNow("未找到 bucket，本地启动可以忽略")
	}
	secretKey := os.Getenv("COS_SECRET_KEY")
	if secretKey == "" {
		s.FailNow("未找到 secretKey，本地启动可以忽略")
	}
	secretID := os.Getenv("COS_SECRET_ID")
	if secretID == "" {
		s.FailNow("未找到 secretID，本地启动可以忽略")
	}
	region := "ap-nanjing"
	s.handler = web.NewHandler(secretID, secretKey, appID,
		bucket, region)
}

func (s *HandlerTestSuite) TestTmpAuthCode() {
	res, err := s.handler.TempAuthCode(&ginx.Context{})
	require.NoError(s.T(), err)
	// 断言有值就可以了
	assert.NotEmpty(s.T(), res.Data.(web.COSTmpAuthCode).SecretKey)
}

func TestHandler(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}
