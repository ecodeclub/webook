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
	"net/http"
	"testing"

	"github.com/ecodeclub/ecache"
	"github.com/ecodeclub/webook/internal/label/internal/integration/startup"
	"github.com/ecodeclub/webook/internal/label/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/label/internal/web"
	"github.com/ecodeclub/webook/internal/test"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/ego-component/egorm"
	"github.com/gotomicro/ego/core/econf"
	"github.com/gotomicro/ego/server/egin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type HandlerTestSuite struct {
	suite.Suite
	server *egin.Component
	db     *egorm.Component
	rdb    ecache.Cache
	dao    dao.LabelDAO
}

func (s *HandlerTestSuite) SetupSuite() {
	handler, err := startup.InitHandler()
	require.NoError(s.T(), err)

	econf.Set("server", map[string]any{"contextTimeout": "1s"})
	server := egin.Load("server").Build()
	handler.PrivateRoutes(server.Engine)

	s.server = server
	s.db = testioc.InitDB()
	err = dao.InitTables(s.db)
	require.NoError(s.T(), err)
	s.dao = dao.NewLabelGORMDAO(s.db)
	s.rdb = testioc.InitCache()
}

func (s *HandlerTestSuite) TearDownSuite() {
	err := s.db.Exec("TRUNCATE TABLE `labels`").Error
	require.NoError(s.T(), err)
}

func (s *HandlerTestSuite) TestSystemLabels() {
	testCases := []struct {
		name   string
		before func(t *testing.T)

		wantCode int
		wantResp test.Result[[]web.Label]
	}{
		{
			name: "查找成功",
			before: func(t *testing.T) {
				err := s.db.Create([]dao.Label{
					{Id: 1, Name: "test", Uid: -1},
					{Id: 2, Name: "non-system", Uid: 123}}).Error
				require.NoError(t, err)
			},
			wantCode: 200,
			wantResp: test.Result[[]web.Label]{
				Data: []web.Label{
					{Id: 1, Name: "test"},
				},
			},
		},
	}
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodGet,
				"/label/system", nil)
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[[]web.Label]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
		})
	}
}

func TestHandler(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}
