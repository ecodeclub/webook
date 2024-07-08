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
	"time"

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
}

func (s *HandlerTestSuite) TestDetail() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	db := s.db.WithContext(ctx)
	// 插入数据的
	err := db.Create(&dao.Roadmap{
		Id:    1,
		Title: "标题1",
		Biz:   sqlx.NewNullString(domain.BizQuestion),
		BizId: sqlx.NewNullInt64(123),
		Ctime: 222,
		Utime: 222,
	}).Error
	require.NoError(s.T(), err)
	edges := []dao.Edge{
		{Id: 1, Rid: 1, SrcBiz: domain.BizQuestionSet, SrcId: 1, DstBiz: domain.BizQuestion, DstId: 2},
		{Id: 2, Rid: 1, SrcBiz: domain.BizQuestion, SrcId: 2, DstBiz: domain.BizQuestionSet, DstId: 3},
		{Id: 3, Rid: 2, SrcBiz: domain.BizQuestion, SrcId: 2, DstBiz: domain.BizQuestionSet, DstId: 3},
	}
	err = db.Create(&edges).Error
	require.NoError(s.T(), err)

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
					Title:    "标题1",
					Biz:      domain.BizQuestion,
					BizId:    123,
					BizTitle: "题目123",
					Utime:    222,
					Edges: []web.Edge{
						{
							Id: 1,
							Src: web.Node{
								Biz:   domain.BizQuestionSet,
								BizId: 1,
								Title: "题集1",
							},
							Dst: web.Node{
								Biz:   domain.BizQuestion,
								BizId: 2,
								Title: "题目2",
							},
						},
						{
							Id: 2,
							Src: web.Node{
								BizId: 2,
								Biz:   domain.BizQuestion,
								Title: "题目2",
							},
							Dst: web.Node{
								BizId: 3,
								Biz:   domain.BizQuestionSet,
								Title: "题集3",
							},
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
			assert.Equal(t, tc.wantResp, recorder.MustScan())
		})
	}
}

func TestHandler(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}
