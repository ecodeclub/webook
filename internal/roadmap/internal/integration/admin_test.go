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
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/ecodeclub/ekit/sqlx"

	"github.com/ecodeclub/ekit/iox"
	"github.com/ecodeclub/ekit/slice"
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
	"gorm.io/gorm"
)

type AdminHandlerTestSuite struct {
	suite.Suite
	db            *egorm.Component
	server        *egin.Component
	hdl           *web.AdminHandler
	dao           dao.AdminDAO
	mockQueSetSvc *quemocks.MockQuestionSetService
	mockQueSvc    *quemocks.MockService
}

func (s *AdminHandlerTestSuite) SetupSuite() {
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
	s.hdl = m.AdminHdl

	econf.Set("server", map[string]any{"contextTimeout": "10s"})
	server := egin.Load("server").Build()
	s.hdl.PrivateRoutes(server.Engine)
	s.server = server
	s.db = testioc.InitDB()
	s.dao = dao.NewGORMAdminDAO(s.db)
}

func (s *AdminHandlerTestSuite) TearDownTest() {
	err := s.db.Exec("TRUNCATE TABLE roadmaps").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE roadmap_edges").Error
	require.NoError(s.T(), err)
}

func (s *AdminHandlerTestSuite) TestSave() {
	testCases := []struct {
		name   string
		before func(t *testing.T)
		after  func(t *testing.T)

		req      web.Roadmap
		wantCode int
		wantResp test.Result[int64]
	}{
		{
			name: "新建",
			before: func(t *testing.T) {

			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				r, err := s.dao.GetById(ctx, 1)
				require.NoError(t, err)
				assert.True(t, r.Utime > 0)
				r.Utime = 0
				assert.True(t, r.Ctime > 0)
				r.Ctime = 0
				assert.Equal(t, dao.Roadmap{
					Id:    1,
					Title: "标题1",
					Biz:   sqlx.NewNullString("test"),
					BizId: sqlx.NewNullInt64(123),
				}, r)
			},
			req: web.Roadmap{
				Title: "标题1",
				Biz:   "test",
				BizId: 123,
			},
			wantCode: 200,
			wantResp: test.Result[int64]{
				Data: 1,
			},
		},
		{
			name: "更新",
			before: func(t *testing.T) {
				s.db.Create(&dao.Roadmap{
					Id:    2,
					Title: "老的标题2",
					Biz:   sqlx.NewNullString("test-old"),
					BizId: sqlx.NewNullInt64(124),
					Ctime: 123,
					Utime: 123,
				})
			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				r, err := s.dao.GetById(ctx, 2)
				require.NoError(t, err)
				assert.True(t, r.Utime > 0)
				r.Utime = 0
				assert.Equal(t, dao.Roadmap{
					Id:    2,
					Title: "标题2",
					Biz:   sqlx.NewNullString("test"),
					BizId: sqlx.NewNullInt64(125),
					Ctime: 123,
				}, r)
			},
			req: web.Roadmap{
				Id:    2,
				Title: "标题2",
				Biz:   "test",
				BizId: 125,
			},
			wantCode: 200,
			wantResp: test.Result[int64]{
				Data: 2,
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/roadmap/save", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[int64]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
			tc.after(t)
		})
	}
}

func (s *AdminHandlerTestSuite) TestList() {
	testCases := []struct {
		name   string
		before func(t *testing.T)
		after  func(t *testing.T)

		req      web.Page
		wantCode int
		wantResp test.Result[web.RoadmapListResp]
	}{
		{
			name: "获取成功",
			before: func(t *testing.T) {
				// 在数据库中插入数据
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				err := s.db.WithContext(ctx).Create(dao.Roadmap{
					Id:    1,
					Title: "标题1",
					Biz:   sqlx.NewNullString(domain.BizQuestionSet),
					BizId: sqlx.NewNullInt64(1),
					Utime: 123,
				}).Error
				require.NoError(t, err)
				err = s.db.WithContext(ctx).Create(dao.Roadmap{
					Id:    2,
					Title: "标题2",
					Biz:   sqlx.NewNullString(domain.BizQuestionSet),
					BizId: sqlx.NewNullInt64(2),
					Utime: 123,
				}).Error
				require.NoError(t, err)
				err = s.db.WithContext(ctx).Create(dao.Roadmap{
					Id:    3,
					Title: "标题3",
					Biz:   sqlx.NewNullString(domain.BizQuestion),
					BizId: sqlx.NewNullInt64(3),
					Utime: 123,
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T) {

			},
			req: web.Page{
				Offset: 0,
				Limit:  3,
			},
			wantCode: 200,
			wantResp: test.Result[web.RoadmapListResp]{
				Data: web.RoadmapListResp{
					Total: 3,
					Maps: []web.Roadmap{
						{
							Id:       3,
							Title:    "标题3",
							Biz:      domain.BizQuestion,
							BizId:    3,
							BizTitle: "题目3",
							Utime:    123,
						},
						{
							Id:       2,
							Title:    "标题2",
							Biz:      domain.BizQuestionSet,
							BizId:    2,
							BizTitle: "题集2",
							Utime:    123,
						},
						{
							Id:       1,
							Title:    "标题1",
							Biz:      domain.BizQuestionSet,
							BizId:    1,
							BizTitle: "题集1",
							Utime:    123,
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/roadmap/list", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.RoadmapListResp]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
			tc.after(t)
		})
	}
}

func (s *AdminHandlerTestSuite) TestDetail() {
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

		req      web.IdReq
		wantCode int
		wantResp test.Result[web.Roadmap]
	}{
		{
			name:     "获取成功",
			req:      web.IdReq{Id: 1},
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

func (s *AdminHandlerTestSuite) TestAddEdge() {
	testCases := []struct {
		name   string
		before func(t *testing.T)
		after  func(t *testing.T)

		req      web.AddEdgeReq
		wantCode int
		wantResp test.Result[any]
	}{
		{
			name: "添加成功",
			before: func(t *testing.T) {

			},
			after: func(t *testing.T) {
				var edge dao.Edge
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				err := s.db.WithContext(ctx).Where("rid = ?", 1).First(&edge).Error
				require.NoError(t, err)
				assert.True(t, edge.Ctime > 0)
				edge.Ctime = 0
				assert.True(t, edge.Utime > 0)
				edge.Utime = 0
				assert.Equal(t, dao.Edge{
					Id:     1,
					Rid:    1,
					SrcBiz: domain.BizQuestion,
					SrcId:  123,
					DstBiz: domain.BizQuestionSet,
					DstId:  234,
				}, edge)
			},
			req: web.AddEdgeReq{
				Rid: 1,
				Edge: web.Edge{
					Src: web.Node{
						Biz:   domain.BizQuestion,
						BizId: 123,
					},
					Dst: web.Node{
						Biz:   domain.BizQuestionSet,
						BizId: 234,
					},
				},
			},
			wantCode: 200,
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/roadmap/edge/save", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[any]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
			tc.after(t)
		})
	}
}

func (s *AdminHandlerTestSuite) TestDelete() {
	testCases := []struct {
		name   string
		before func(t *testing.T)
		after  func(t *testing.T)

		req      web.IdReq
		wantCode int
		wantResp test.Result[any]
	}{
		{
			name: "删除成功",
			before: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				err := s.db.WithContext(ctx).Create(&dao.Edge{
					Id: 1,
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				var edge dao.Edge
				err := s.db.WithContext(ctx).Where("id = ?", 1).First(&edge).Error
				assert.Equal(t, gorm.ErrRecordNotFound, err)
			},
			wantCode: 200,
			req:      web.IdReq{Id: 1},
		},
	}
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/roadmap/edge/delete", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[any]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
			tc.after(t)
		})
	}
}

func TestAdminHandler(t *testing.T) {
	suite.Run(t, new(AdminHandlerTestSuite))
}
