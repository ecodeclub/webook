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
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/ecodeclub/ekit/iox"
	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ekit/sqlx"
	baguwen "github.com/ecodeclub/webook/internal/question"
	quemocks "github.com/ecodeclub/webook/internal/question/mocks"
	"github.com/ecodeclub/webook/internal/roadmap/internal/domain"
	"github.com/ecodeclub/webook/internal/roadmap/internal/integration/startup"
	"github.com/ecodeclub/webook/internal/roadmap/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/roadmap/internal/web"
	"github.com/ecodeclub/webook/internal/test"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/ego-component/egorm"
	"github.com/gotomicro/ego/core/econf"
	"github.com/gotomicro/ego/server/egin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type HandlerTestSuite struct {
	suite.Suite
	db     *egorm.Component
	server *egin.Component
	hdl    *web.Handler
	dao    dao.AdminDAO
}

func (s *HandlerTestSuite) SetupSuite() {
	ctrl := gomock.NewController(s.T())
	mockQueSvc := quemocks.NewMockService(ctrl)
	// mockQueSvc 固定返回

	mockQueSvc.EXPECT().GetPubByIDs(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, ids []int64) ([]baguwen.Question, error) {
			return slice.Map(ids, func(idx int, src int64) baguwen.Question {
				return baguwen.Question{
					Id:    src,
					Title: fmt.Sprintf("题目%d", src),
				}
			}), nil
		}).AnyTimes()

	mockQueSetSvc := quemocks.NewMockQuestionSetService(ctrl)
	mockQueSetSvc.EXPECT().GetByIds(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, ids []int64) ([]baguwen.QuestionSet, error) {
			return slice.Map(ids, func(idx int, src int64) baguwen.QuestionSet {
				return baguwen.QuestionSet{
					Id:    src,
					Title: fmt.Sprintf("题集%d", src),
				}
			}), nil
		}).AnyTimes()

	m := startup.InitModule(&baguwen.Module{
		Svc:    mockQueSvc,
		SetSvc: mockQueSetSvc,
	})
	s.hdl = m.Hdl

	econf.Set("server", map[string]any{"contextTimeout": "10s"})
	server := egin.Load("server").Build()
	s.hdl.PrivateRoutes(server.Engine)
	s.server = server
	s.db = testioc.InitDB()
	s.dao = dao.NewGORMAdminDAO(s.db)
}

func (s *HandlerTestSuite) TearDownTest() {
	err := s.db.Exec("TRUNCATE TABLE roadmaps").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE roadmap_edges").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE roadmap_nodes").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE roadmap_edges_v1").Error
	require.NoError(s.T(), err)
}

func (s *HandlerTestSuite) TestDetail() {
	t := s.T()
	// 创建roadmap
	err := s.db.Create(&dao.Roadmap{
		Id:    1,
		Title: "Roadmap 1",
		Biz:   sqlx.NewNullString("question"),
		BizId: sqlx.NewNullInt64(123),
		Ctime: 222,
		Utime: 222,
	}).Error
	require.NoError(t, err)

	// 创建三个节点
	nodes := []dao.Node{
		{Id: 1, Biz: "question", Rid: 1, RefId: 123, Attrs: "attributes1"},
		{Id: 2, Biz: "questionSet", Rid: 1, RefId: 456, Attrs: "attributes2"},
		{Id: 3, Biz: "questionSet", Rid: 1, RefId: 789, Attrs: "attributes3"},
	}
	err = s.db.Create(&nodes).Error
	require.NoError(t, err)

	// 创建三条边
	edges := []dao.EdgeV1{
		{Id: 1, Rid: 1, SrcNode: 1, DstNode: 3, Type: "default", Attrs: "edge attributes 1"},
		{Id: 2, Rid: 1, SrcNode: 3, DstNode: 2, Type: "default", Attrs: "edge attributes 2"},
		{Id: 3, Rid: 2, SrcNode: 3, DstNode: 2, Type: "default", Attrs: "edge attributes 3"},
	}
	err = s.db.Create(&edges).Error
	require.NoError(t, err)

	testCases := []struct {
		name string

		req      web.Biz
		wantCode int
		wantResp test.Result[web.Roadmap]
	}{
		{
			name:     "获取成功",
			req:      web.Biz{BizId: 123, Biz: domain.BizQuestion},
			wantCode: 200,
			wantResp: test.Result[web.Roadmap]{
				Data: web.Roadmap{
					Id:       1,
					Title:    "Roadmap 1",
					Biz:      "question",
					BizId:    123,
					BizTitle: "题目123",
					Utime:    222,
					Edges: []web.Edge{
						{
							Id: 2,
							Src: web.Node{
								ID:    3,
								Biz:   "questionSet",
								BizId: 789,
								Rid:   1,
								Attrs: "attributes3",
								Title: "题集789",
							},
							Dst: web.Node{
								ID:    2,
								Biz:   "questionSet",
								BizId: 456,
								Rid:   1,
								Attrs: "attributes2",
								Title: "题集456",
							},
							Type:  "default",
							Attrs: "edge attributes 2",
						},
						{
							Id: 1,
							Src: web.Node{
								ID:    1,
								Biz:   "question",
								BizId: 123,
								Rid:   1,
								Attrs: "attributes1",
								Title: "题目123",
							},
							Dst: web.Node{
								ID:    3,
								Biz:   "questionSet",
								BizId: 789,
								Rid:   1,
								Attrs: "attributes3",
								Title: "题集789",
							},
							Type:  "default",
							Attrs: "edge attributes 1",
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost,
				"/roadmap/detail", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.Roadmap]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			data := recorder.MustScan()
			assert.Equal(t, tc.wantResp, data)
		})
	}
}

func TestHandler(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}
