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
	"net/http/httptest"
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
	err = s.db.Exec("TRUNCATE TABLE roadmap_nodes").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE roadmap_edges_v1").Error
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
	testCases := []struct {
		name     string
		before   func(t *testing.T)
		after    func(t *testing.T)
		req      web.IdReq
		wantCode int
		wantResp test.Result[web.Roadmap]
	}{
		{
			name: "获取成功",
			before: func(t *testing.T) {
				// 创建roadmap
				err := s.db.Create(&dao.Roadmap{
					Id:    1,
					Title: "Roadmap 1",
					Biz:   sqlx.NewNullString("question"),
					BizId: sqlx.NewNullInt64(123),
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
			},
			after: func(t *testing.T) {
				// 清理数据库或其他后置操作
			},
			req:      web.IdReq{Id: 1},
			wantCode: 200,
			wantResp: test.Result[web.Roadmap]{
				Data: web.Roadmap{
					Id:       1,
					Title:    "Roadmap 1",
					Biz:      "question",
					BizId:    123,
					BizTitle: "题目123",
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
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/roadmap/detail", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.Roadmap]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
			tc.after(t)
		})
	}
}

func (s *AdminHandlerTestSuite) TestSaveEdge() {
	testCases := []struct {
		name     string
		before   func(t *testing.T)
		after    func(t *testing.T)
		req      web.AddEdgeReq
		wantCode int
		wantResp test.Result[any]
	}{
		{
			name: "新增边成功",
			before: func(t *testing.T) {
				// 创建三个节点
				nodes := []dao.Node{
					{Id: 1, Biz: "question", Rid: 1, RefId: 123, Attrs: "attributes1"},
					{Id: 2, Biz: "case", Rid: 1, RefId: 456, Attrs: "attributes2"},
					{Id: 3, Biz: "common", Rid: 0, RefId: 789, Attrs: "attributes3"},
				}
				err := s.db.Create(&nodes).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				// 验证边已被添加
				var edge dao.EdgeV1
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				err := s.db.WithContext(ctx).Where("src_node = ? AND dst_node = ?", 1, 2).First(&edge).Error
				require.NoError(t, err)
				require.True(t, edge.Ctime != 0)
				require.True(t, edge.Utime != 0)
				edge.Ctime = 0
				edge.Utime = 0
				assert.Equal(t, dao.EdgeV1{
					Id:      1,
					SrcNode: 1,
					DstNode: 2,
					Rid:     1,
					Type:    "default",
					Attrs:   "attrs",
				}, edge)
			},
			req: web.AddEdgeReq{
				Rid: 1,
				Edge: web.Edge{
					Id:    1,
					Src:   web.Node{ID: 1},
					Dst:   web.Node{ID: 2},
					Type:  "default",
					Attrs: "attrs",
				},
			},
			wantCode: 200,
			wantResp: test.Result[any]{},
		},
		{
			name: "编辑边成功",
			before: func(t *testing.T) {
				// 创建一个边
				nodes := []dao.Node{
					{Id: 1, Biz: "question", Rid: 1, RefId: 123, Attrs: "attributes1"},
					{Id: 2, Biz: "case", Rid: 1, RefId: 456, Attrs: "attributes2"},
					{Id: 3, Biz: "common", Rid: 0, RefId: 789, Attrs: "attributes3"},
				}
				err := s.db.Create(&nodes).Error
				require.NoError(t, err)
				err = s.db.Create(&dao.EdgeV1{
					Id:      1,
					SrcNode: 1,
					DstNode: 2,
					Rid:     1,
					Type:    "default",
					Attrs:   "attrs",
					Ctime:   123,
					Utime:   321,
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				// 验证边已被编辑
				var edge dao.EdgeV1
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				err := s.db.WithContext(ctx).Where("id = ?", 1).First(&edge).Error
				require.NoError(t, err)
				require.True(t, edge.Ctime != 0)
				require.True(t, edge.Utime != 0)
				edge.Ctime = 0
				edge.Utime = 0
				assert.Equal(t, dao.EdgeV1{
					Id:      1,
					SrcNode: 1,
					DstNode: 3, // 更新后的目标节点
					Rid:     1,
					Type:    "updated",
					Attrs:   "attrsv1",
				}, edge)
			},
			req: web.AddEdgeReq{
				Rid: 1,
				Edge: web.Edge{
					Id:    1, // 指定边的ID以进行编辑
					Src:   web.Node{ID: 1},
					Dst:   web.Node{ID: 3}, // 更新目标节点
					Type:  "updated",
					Attrs: "attrsv1",
				},
			},
			wantCode: 200,
			wantResp: test.Result[any]{},
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
			err = s.db.Exec("TRUNCATE TABLE roadmap_nodes").Error
			require.NoError(s.T(), err)
			err = s.db.Exec("TRUNCATE TABLE roadmap_edges_v1").Error
			require.NoError(s.T(), err)
		})
	}
}

func (s *AdminHandlerTestSuite) TestDeleteEdge() {
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
				err := s.db.WithContext(ctx).Create(&dao.EdgeV1{
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

func (s *AdminHandlerTestSuite) TestSaveNode() {
	testCases := []struct {
		name     string
		before   func(t *testing.T)
		after    func(t *testing.T)
		req      web.Node
		wantCode int
		wantResp test.Result[int64]
	}{
		{
			name: "新建节点成功",
			before: func(t *testing.T) {
				// 可以在这里设置测试前的数据库状态或其他依赖
			},
			after: func(t *testing.T) {
				var node dao.Node
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				err := s.db.WithContext(ctx).Where("id = ?", 1).First(&node).Error
				require.NoError(t, err)
				assert.True(t, node.Ctime > 0)
				node.Ctime = 0
				assert.True(t, node.Utime > 0)
				node.Utime = 0
				assert.Equal(t, dao.Node{
					Id:    1,
					Biz:   "question",
					Rid:   1,
					RefId: 123,
					Attrs: "some attributes",
				}, node)
			},
			req: web.Node{
				Biz:   "question",
				Rid:   1,
				BizId: 123,
				Attrs: "some attributes",
			},
			wantCode: 200,
			wantResp: test.Result[int64]{Data: 1},
		},
		{
			name: "更新节点成功",
			before: func(t *testing.T) {
				err := s.db.Create(&dao.Node{
					Id:    2,
					Biz:   "question",
					Rid:   2,
					RefId: 456,
					Attrs: "old attributes",
					Ctime: 123,
					Utime: 123,
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				var node dao.Node
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				err := s.db.WithContext(ctx).Where("id = ?", 2).First(&node).Error
				require.NoError(t, err)
				assert.True(t, node.Utime > 123)
				node.Utime = 0
				assert.Equal(t, dao.Node{
					Id:    2,
					Biz:   "case",
					Rid:   2,
					RefId: 789,
					Attrs: "new attributes",
					Ctime: 123,
				}, node)
			},
			req: web.Node{
				ID:    2,
				Biz:   "case",
				Rid:   2,
				BizId: 789,
				Attrs: "new attributes",
			},
			wantCode: 200,
			wantResp: test.Result[int64]{Data: 2},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/roadmap/node/save", iox.NewJSONReader(tc.req))
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

func (s *AdminHandlerTestSuite) TestDeleteNode() {
	testCases := []struct {
		name     string
		before   func(t *testing.T)
		after    func(t *testing.T)
		req      web.IdReq
		wantCode int
		wantResp test.Result[any]
	}{
		{
			name: "删除节点成功",
			before: func(t *testing.T) {
				// 预先插入一个节点
				err := s.db.Create(&dao.Node{
					Id:    1,
					Biz:   "question",
					Rid:   1,
					RefId: 123,
					Attrs: "some attributes",
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				// 验证节点已被删除
				var node dao.Node
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				err := s.db.WithContext(ctx).Where("id = ?", 1).First(&node).Error
				assert.Equal(t, gorm.ErrRecordNotFound, err)
			},
			req:      web.IdReq{Id: 1},
			wantCode: 200,
			wantResp: test.Result[any]{},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/roadmap/node/delete", iox.NewJSONReader(tc.req))
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

func (s *AdminHandlerTestSuite) TestNodeList() {
	testCases := []struct {
		name     string
		before   func(t *testing.T)
		after    func(t *testing.T)
		req      web.IdReq
		wantCode int
		wantResp test.Result[[]web.Node]
	}{
		{
			name: "获取节点列表成功，包括rid为0和rid为3的节点",
			before: func(t *testing.T) {
				// 预先插入一些节点，包括rid为0和rid为3的节点
				nodes := []dao.Node{
					{Id: 1, Biz: "question", Rid: 1, RefId: 123, Attrs: "attributes1"},
					{Id: 2, Biz: "case", Rid: 1, RefId: 456, Attrs: "attributes2"},
					{Id: 3, Biz: "common", Rid: 0, RefId: 789, Attrs: "attributes3"},  // rid为0的节点
					{Id: 4, Biz: "special", Rid: 3, RefId: 101, Attrs: "attributes4"}, // rid为3的节点
				}
				err := s.db.Create(&nodes).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				// 清理数据库或其他后置操作
			},
			req:      web.IdReq{Id: 1},
			wantCode: 200,
			wantResp: test.Result[[]web.Node]{
				Data: []web.Node{
					{ID: 3, Biz: "common", Rid: 0, BizId: 789, Attrs: "attributes3"},
					{ID: 2, Biz: "case", Rid: 1, BizId: 456, Attrs: "attributes2"},
					{ID: 1, Biz: "question", Rid: 1, BizId: 123, Attrs: "attributes1"},
				},
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/roadmap/node/list", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[[]web.Node]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
			tc.after(t)
		})
	}
}

func (s *AdminHandlerTestSuite) TestSanitize() {
	roadmaps := []dao.Roadmap{
		{Id: 1, Title: "Roadmap 1", Biz: sqlx.NewNullString("biz1"), BizId: sqlx.NewNullInt64(101)},
		{Id: 2, Title: "Roadmap 2", Biz: sqlx.NewNullString("biz2"), BizId: sqlx.NewNullInt64(102)},
		{Id: 3, Title: "Roadmap 3", Biz: sqlx.NewNullString("biz3"), BizId: sqlx.NewNullInt64(103)},
	}
	for _, roadmap := range roadmaps {
		err := s.db.Create(&roadmap).Error
		require.NoError(s.T(), err)
	}
	edges := []dao.Edge{
		{Rid: 1, SrcBiz: "biz1", SrcId: 1, DstBiz: "biz1", DstId: 2},
		{Rid: 1, SrcBiz: "biz1", SrcId: 2, DstBiz: "biz1", DstId: 3},
		{Rid: 1, SrcBiz: "biz1", SrcId: 3, DstBiz: "biz1", DstId: 4},
		{Rid: 1, SrcBiz: "biz1", SrcId: 4, DstBiz: "biz1", DstId: 5},
		{Rid: 2, SrcBiz: "biz2", SrcId: 1, DstBiz: "biz2", DstId: 2},
		{Rid: 2, SrcBiz: "biz2", SrcId: 2, DstBiz: "biz2", DstId: 3},
		{Rid: 2, SrcBiz: "biz2", SrcId: 3, DstBiz: "biz2", DstId: 4},
		{Rid: 2, SrcBiz: "biz2", SrcId: 4, DstBiz: "biz2", DstId: 5},
		{Rid: 3, SrcBiz: "biz3", SrcId: 1, DstBiz: "biz3", DstId: 2},
		{Rid: 3, SrcBiz: "biz3", SrcId: 2, DstBiz: "biz3", DstId: 3},
		{Rid: 3, SrcBiz: "biz3", SrcId: 3, DstBiz: "biz3", DstId: 4},
		{Rid: 3, SrcBiz: "biz3", SrcId: 4, DstBiz: "biz3", DstId: 5},
	}
	for _, edge := range edges {
		err := s.db.Create(&edge).Error
		require.NoError(s.T(), err)
	}
	req, err := http.NewRequest(http.MethodPost, "/roadmap/sanitize", nil)
	require.NoError(s.T(), err)
	recorder := httptest.NewRecorder()
	s.server.ServeHTTP(recorder, req)
	require.Equal(s.T(), http.StatusOK, recorder.Code)

	time.Sleep(10 * time.Second)
	s.checkSanitizeData(edges)
}

func (s *AdminHandlerTestSuite) checkSanitizeData(edge1s []dao.Edge) {
	var nodes []dao.Node
	err := s.db.WithContext(context.Background()).Model(&dao.Node{}).Find(&nodes).Error
	require.NoError(s.T(), err)
	nodeMap := s.getNodeMap(nodes)
	wantEdgev1s := slice.Map(edge1s, func(idx int, src dao.Edge) dao.EdgeV1 {
		return s.getEdgev1(src, nodeMap)
	})
	var edgev1s []dao.EdgeV1
	err = s.db.WithContext(context.Background()).Model(&dao.EdgeV1{}).Find(&edgev1s).Error
	require.NoError(s.T(), err)
	actualEdgev1s := slice.Map(edgev1s, func(idx int, src dao.EdgeV1) dao.EdgeV1 {
		require.True(s.T(), src.Ctime != 0)
		require.True(s.T(), src.Utime != 0)
		require.True(s.T(), src.Id != 0)
		src.Ctime = 0
		src.Utime = 0
		src.Id = 0
		return src
	})
	assert.ElementsMatch(s.T(), wantEdgev1s, actualEdgev1s)
}

func (s *AdminHandlerTestSuite) getEdgev1(edge dao.Edge, nodeMap map[string]dao.Node) dao.EdgeV1 {
	dstNode := nodeMap[fmt.Sprintf("%d_%s_%d", edge.Rid, edge.DstBiz, edge.DstId)]
	srcNode := nodeMap[fmt.Sprintf("%d_%s_%d", edge.Rid, edge.SrcBiz, edge.SrcId)]
	return dao.EdgeV1{
		Rid:     edge.Rid,
		SrcNode: srcNode.Id,
		DstNode: dstNode.Id,
	}
}

func (s *AdminHandlerTestSuite) getNodeMap(nodes []dao.Node) map[string]dao.Node {
	nodeMap := make(map[string]dao.Node, len(nodes))
	for _, node := range nodes {
		nodeMap[fmt.Sprintf("%d_%s_%d", node.Rid, node.Biz, node.RefId)] = node
	}
	return nodeMap
}

func TestAdminHandler(t *testing.T) {
	suite.Run(t, new(AdminHandlerTestSuite))
}
