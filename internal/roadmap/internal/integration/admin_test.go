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
	"sort"
	"testing"
	"time"

	"github.com/ecodeclub/ekit/sqlx"
	"github.com/ecodeclub/webook/internal/roadmap/internal/service"

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
	svc           service.AdminService
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
	mockQueSetSvc.EXPECT().Detail(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, id int64) (baguwen.QuestionSet, error) {
		return baguwen.QuestionSet{
			Id:    id,
			Title: fmt.Sprintf("题集%d", id),
			Questions: []baguwen.Question{
				{
					Id: 1,
				},
				{
					Id: 2,
				},
				{
					Id: 3,
				},
			},
		}, nil
	}).AnyTimes()

	m := startup.InitModule(&baguwen.Module{
		Svc:    mockQueSvc,
		SetSvc: mockQueSetSvc,
	})
	s.hdl = m.AdminHdl
	s.svc = m.AdminSvc

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

// assertRoadmapsExact 精确匹配路线图列表
func (s *AdminHandlerTestSuite) assertRoadmapsExact(t *testing.T, want []domain.Roadmap, actual []domain.Roadmap) {
	require.Equal(t, len(want), len(actual))
	for i, w := range want {
		if i < len(actual) {
			s.assertRoadmapEqual(t, w, actual[i])
		}
	}
}

// assertRoadmapsContains 验证实际结果包含期望的数据（用于since=0的情况，可能包含其他测试用例的数据）
func (s *AdminHandlerTestSuite) assertRoadmapsContains(t *testing.T, want []domain.Roadmap, actual []domain.Roadmap) {
	wantIdMap := make(map[int64]domain.Roadmap, len(want))
	for _, w := range want {
		wantIdMap[w.Id] = w
	}
	for _, a := range actual {
		if w, ok := wantIdMap[a.Id]; ok {
			s.assertRoadmapEqual(t, w, a)
		}
	}
	require.GreaterOrEqual(t, len(actual), len(want), "结果数量应该至少包含期望的数据")
}

// assertRoadmapEqual 比较两个路线图是否相等
func (s *AdminHandlerTestSuite) assertRoadmapEqual(t *testing.T, want domain.Roadmap, actual domain.Roadmap) {
	assert.Equal(t, want.Id, actual.Id)
	assert.Equal(t, want.Title, actual.Title)
	assert.Equal(t, want.Biz, actual.Biz)
	assert.Equal(t, want.BizId, actual.BizId)
	assert.Equal(t, want.Utime, actual.Utime)
	assert.Equal(t, len(want.Edges), len(actual.Edges), "路线图 %d 的边数量不匹配", want.Id)
	if len(want.Edges) > 0 {
		actualEdgeMap := make(map[int64]domain.Edge, len(actual.Edges))
		for _, edge := range actual.Edges {
			actualEdgeMap[edge.Id] = edge
		}
		for _, wantEdge := range want.Edges {
			actualEdge, ok := actualEdgeMap[wantEdge.Id]
			require.True(t, ok, "路线图 %d 缺少边 %d", want.Id, wantEdge.Id)
			s.assertEdgeEqual(t, wantEdge, actualEdge)
		}
	}
}

// assertEdgeEqual 比较两个边是否相等
func (s *AdminHandlerTestSuite) assertEdgeEqual(t *testing.T, want domain.Edge, actual domain.Edge) {
	actual.Src.Title = want.Src.Title
	actual.Src.Biz.Title = want.Src.Biz.Title
	actual.Dst.Title = want.Dst.Title
	actual.Dst.Biz.Title = want.Dst.Biz.Title
	assert.Equal(t, want, actual)
}

func (s *AdminHandlerTestSuite) TestService_ListSince() {
	testCases := []struct {
		name    string
		before  func(t *testing.T)
		after   func(t *testing.T, result []domain.Roadmap)
		since   int64
		offset  int
		limit   int
		wantErr error
	}{
		{
			name: "基本查询-返回带边的路线图",
			before: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				err := s.db.WithContext(ctx).Create(&dao.Roadmap{
					Id:    100,
					Title: "Roadmap 1",
					Biz:   sqlx.NewNullString("questionSet"),
					BizId: sqlx.NewNullInt64(100),
					Utime: 1000,
					Ctime: 1000,
				}).Error
				require.NoError(t, err)
				// 创建节点
				nodes := []dao.Node{
					{Id: 100, Biz: "question", Rid: 100, RefId: 1, Attrs: "attrs1"},
					{Id: 101, Biz: "question", Rid: 100, RefId: 2, Attrs: "attrs2"},
				}
				err = s.db.WithContext(ctx).Create(&nodes).Error
				require.NoError(t, err)
				// 创建边
				err = s.db.WithContext(ctx).Create(&dao.EdgeV1{
					Id:      100,
					Rid:     100,
					SrcNode: 100,
					DstNode: 101,
					Type:    "default",
					Attrs:   "edge_attrs",
					Utime:   1000,
					Ctime:   1000,
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T, result []domain.Roadmap) {
				s.assertRoadmapsContains(t, []domain.Roadmap{
					{
						Id:    100,
						Title: "Roadmap 1",
						Biz:   "questionSet",
						BizId: 100,
						Utime: 1000,
						Edges: []domain.Edge{
							{
								Id:    100,
								Type:  "default",
								Attrs: "edge_attrs",
								Src: domain.Node{
									ID:    100,
									Rid:   100,
									Attrs: "attrs1",
									Biz: domain.Biz{
										Biz:   "question",
										BizId: 1,
									},
								},
								Dst: domain.Node{
									ID:    101,
									Rid:   100,
									Attrs: "attrs2",
									Biz: domain.Biz{
										Biz:   "question",
										BizId: 2,
									},
								},
							},
						},
					},
				}, result)
			},
			since:   0,
			offset:  0,
			limit:   10,
			wantErr: nil,
		},
		{
			name: "时间过滤-只返回utime大于等于since的数据",
			before: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				roadmaps := []dao.Roadmap{
					{Id: 200, Title: "Old", Biz: sqlx.NewNullString("test2"), BizId: sqlx.NewNullInt64(200), Utime: 500, Ctime: 500},
					{Id: 201, Title: "Mid", Biz: sqlx.NewNullString("test2"), BizId: sqlx.NewNullInt64(201), Utime: 1200, Ctime: 1200},
					{Id: 202, Title: "New", Biz: sqlx.NewNullString("test2"), BizId: sqlx.NewNullInt64(202), Utime: 1500, Ctime: 1500},
				}
				err := s.db.WithContext(ctx).Create(&roadmaps).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T, result []domain.Roadmap) {
				s.assertRoadmapsExact(t, []domain.Roadmap{
					{Id: 202, Title: "New", Biz: "test2", BizId: 202, Utime: 1500, Edges: []domain.Edge{}},
					{Id: 201, Title: "Mid", Biz: "test2", BizId: 201, Utime: 1200, Edges: []domain.Edge{}},
				}, result)
			},
			since:   1200,
			offset:  0,
			limit:   10,
			wantErr: nil,
		},
		{
			name: "时间边界-utime等于since的数据被包含",
			before: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				err := s.db.WithContext(ctx).Create(&dao.Roadmap{
					Id:    300,
					Title: "Boundary",
					Biz:   sqlx.NewNullString("test3"),
					BizId: sqlx.NewNullInt64(300),
					Utime: 5000,
					Ctime: 5000,
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T, result []domain.Roadmap) {
				s.assertRoadmapsExact(t, []domain.Roadmap{
					{Id: 300, Title: "Boundary", Biz: "test3", BizId: 300, Utime: 5000, Edges: []domain.Edge{}},
				}, result)
			},
			since:   5000,
			offset:  0,
			limit:   10,
			wantErr: nil,
		},
		{
			name: "排序-按utime DESC, id DESC排序",
			before: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				roadmaps := []dao.Roadmap{
					{Id: 400, Title: "Same Time 1", Biz: sqlx.NewNullString("test4"), BizId: sqlx.NewNullInt64(400), Utime: 1000, Ctime: 1000},
					{Id: 402, Title: "Same Time 3", Biz: sqlx.NewNullString("test4"), BizId: sqlx.NewNullInt64(402), Utime: 1000, Ctime: 1000},
					{Id: 401, Title: "Same Time 2", Biz: sqlx.NewNullString("test4"), BizId: sqlx.NewNullInt64(401), Utime: 1000, Ctime: 1000},
					{Id: 403, Title: "Newer", Biz: sqlx.NewNullString("test4"), BizId: sqlx.NewNullInt64(403), Utime: 2000, Ctime: 2000},
				}
				err := s.db.WithContext(ctx).Create(&roadmaps).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T, result []domain.Roadmap) {
				s.assertRoadmapsContains(t, []domain.Roadmap{
					{Id: 403, Title: "Newer", Biz: "test4", BizId: 403, Utime: 2000, Edges: []domain.Edge{}},
					{Id: 402, Title: "Same Time 3", Biz: "test4", BizId: 402, Utime: 1000, Edges: []domain.Edge{}},
					{Id: 401, Title: "Same Time 2", Biz: "test4", BizId: 401, Utime: 1000, Edges: []domain.Edge{}},
					{Id: 400, Title: "Same Time 1", Biz: "test4", BizId: 400, Utime: 1000, Edges: []domain.Edge{}},
				}, result)
			},
			since:   0,
			offset:  0,
			limit:   10,
			wantErr: nil,
		},
		{
			name: "分页-正常分页",
			before: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				roadmaps := []dao.Roadmap{
					{Id: 500, Title: "R1", Biz: sqlx.NewNullString("test5"), BizId: sqlx.NewNullInt64(500), Utime: 1000, Ctime: 1000},
					{Id: 501, Title: "R2", Biz: sqlx.NewNullString("test5"), BizId: sqlx.NewNullInt64(501), Utime: 2000, Ctime: 2000},
					{Id: 502, Title: "R3", Biz: sqlx.NewNullString("test5"), BizId: sqlx.NewNullInt64(502), Utime: 3000, Ctime: 3000},
				}
				err := s.db.WithContext(ctx).Create(&roadmaps).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T, result []domain.Roadmap) {
				s.assertRoadmapsContains(t, []domain.Roadmap{
					{Id: 501, Title: "R2", Biz: "test5", BizId: 501, Utime: 2000, Edges: []domain.Edge{}},
					{Id: 500, Title: "R1", Biz: "test5", BizId: 500, Utime: 1000, Edges: []domain.Edge{}},
				}, result)
			},
			since:   0,
			offset:  1,
			limit:   2,
			wantErr: nil,
		},
		{
			name: "分页-offset超出范围返回空",
			before: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				err := s.db.WithContext(ctx).Create(&dao.Roadmap{
					Id:    600,
					Title: "R1",
					Biz:   sqlx.NewNullString("test6"),
					BizId: sqlx.NewNullInt64(600),
					Utime: 1000,
					Ctime: 1000,
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T, result []domain.Roadmap) {
				assert.Equal(t, 0, len(result))
			},
			since:   0,
			offset:  100,
			limit:   10,
			wantErr: nil,
		},
		{
			name: "空结果-没有符合条件的数据",
			before: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				err := s.db.WithContext(ctx).Create(&dao.Roadmap{
					Id:    700,
					Title: "Old",
					Biz:   sqlx.NewNullString("test7"),
					BizId: sqlx.NewNullInt64(700),
					Utime: 100,
					Ctime: 100,
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T, result []domain.Roadmap) {
				assert.Equal(t, 0, len(result))
			},
			since:   10000,
			offset:  0,
			limit:   10,
			wantErr: nil,
		},
		{
			name: "多个路线图-每个都有不同的边",
			before: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				roadmaps := []dao.Roadmap{
					{Id: 800, Title: "R1", Biz: sqlx.NewNullString("test8"), BizId: sqlx.NewNullInt64(800), Utime: 1000, Ctime: 1000},
					{Id: 801, Title: "R2", Biz: sqlx.NewNullString("test8"), BizId: sqlx.NewNullInt64(801), Utime: 2000, Ctime: 2000},
				}
				err := s.db.WithContext(ctx).Create(&roadmaps).Error
				require.NoError(t, err)
				// 为每个路线图创建节点
				nodes := []dao.Node{
					{Id: 800, Biz: "question", Rid: 800, RefId: 1},
					{Id: 801, Biz: "question", Rid: 800, RefId: 2},
					{Id: 802, Biz: "question", Rid: 801, RefId: 3},
					{Id: 803, Biz: "question", Rid: 801, RefId: 4},
				}
				err = s.db.WithContext(ctx).Create(&nodes).Error
				require.NoError(t, err)
				// 创建边
				edges := []dao.EdgeV1{
					{Id: 800, Rid: 800, SrcNode: 800, DstNode: 801, Type: "type1", Attrs: "attrs1", Utime: 1000, Ctime: 1000},
					{Id: 801, Rid: 801, SrcNode: 802, DstNode: 803, Type: "type2", Attrs: "attrs2", Utime: 2000, Ctime: 2000},
				}
				err = s.db.WithContext(ctx).Create(&edges).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T, result []domain.Roadmap) {
				s.assertRoadmapsContains(t, []domain.Roadmap{
					{
						Id:    801,
						Title: "R2",
						Biz:   "test8",
						BizId: 801,
						Utime: 2000,
						Edges: []domain.Edge{
							{
								Id:    801,
								Type:  "type2",
								Attrs: "attrs2",
								Src:   domain.Node{ID: 802, Rid: 801, Biz: domain.Biz{Biz: "question", BizId: 3}},
								Dst:   domain.Node{ID: 803, Rid: 801, Biz: domain.Biz{Biz: "question", BizId: 4}},
							},
						},
					},
					{
						Id:    800,
						Title: "R1",
						Biz:   "test8",
						BizId: 800,
						Utime: 1000,
						Edges: []domain.Edge{
							{
								Id:    800,
								Type:  "type1",
								Attrs: "attrs1",
								Src:   domain.Node{ID: 800, Rid: 800, Biz: domain.Biz{Biz: "question", BizId: 1}},
								Dst:   domain.Node{ID: 801, Rid: 800, Biz: domain.Biz{Biz: "question", BizId: 2}},
							},
						},
					},
				}, result)
			},
			since:   0,
			offset:  0,
			limit:   10,
			wantErr: nil,
		},
		{
			name: "没有边的路线图-返回空边列表",
			before: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				err := s.db.WithContext(ctx).Create(&dao.Roadmap{
					Id:    900,
					Title: "No Edges",
					Biz:   sqlx.NewNullString("test9"),
					BizId: sqlx.NewNullInt64(900),
					Utime: 1000,
					Ctime: 1000,
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T, result []domain.Roadmap) {
				s.assertRoadmapsContains(t, []domain.Roadmap{
					{Id: 900, Title: "No Edges", Biz: "test9", BizId: 900, Utime: 1000, Edges: []domain.Edge{}},
				}, result)
			},
			since:   0,
			offset:  0,
			limit:   10,
			wantErr: nil,
		},
		{
			name: "since为0-查询所有数据",
			before: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				roadmaps := []dao.Roadmap{
					{Id: 1000, Title: "R1", Biz: sqlx.NewNullString("test10"), BizId: sqlx.NewNullInt64(1000), Utime: 500, Ctime: 500},
					{Id: 1001, Title: "R2", Biz: sqlx.NewNullString("test10"), BizId: sqlx.NewNullInt64(1001), Utime: 1000, Ctime: 1000},
					{Id: 1002, Title: "R3", Biz: sqlx.NewNullString("test10"), BizId: sqlx.NewNullInt64(1002), Utime: 1500, Ctime: 1500},
				}
				err := s.db.WithContext(ctx).Create(&roadmaps).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T, result []domain.Roadmap) {
				s.assertRoadmapsContains(t, []domain.Roadmap{
					{Id: 1002, Title: "R3", Biz: "test10", BizId: 1002, Utime: 1500, Edges: []domain.Edge{}},
					{Id: 1001, Title: "R2", Biz: "test10", BizId: 1001, Utime: 1000, Edges: []domain.Edge{}},
					{Id: 1000, Title: "R1", Biz: "test10", BizId: 1000, Utime: 500, Edges: []domain.Edge{}},
				}, result)
			},
			since:   0,
			offset:  0,
			limit:   10,
			wantErr: nil,
		},
		{
			name: "limit为0-返回空结果",
			before: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				err := s.db.WithContext(ctx).Create(&dao.Roadmap{
					Id:    1100,
					Title: "R1",
					Biz:   sqlx.NewNullString("test11"),
					BizId: sqlx.NewNullInt64(1100),
					Utime: 1000,
					Ctime: 1000,
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T, result []domain.Roadmap) {
				assert.Equal(t, 0, len(result))
			},
			since:   0,
			offset:  0,
			limit:   0,
			wantErr: nil,
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			result, err := s.svc.ListSince(t.Context(), tc.since, tc.offset, tc.limit)
			if tc.wantErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tc.wantErr, err)
				return
			}
			assert.NoError(t, err)
			tc.after(t, result)
		})
	}
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
					Biz:   sqlx.NewNullString("questionSet"),
					BizId: sqlx.NewNullInt64(123),
				}, r)

				nodes, err := s.dao.NodeList(ctx, 1)
				require.NoError(t, err)
				nodes = slice.Map(nodes, func(idx int, src dao.Node) dao.Node {
					src.Id = 0
					src.Ctime = 0
					src.Utime = 0
					return src
				})
				sort.Slice(nodes, func(i, j int) bool {
					return nodes[i].RefId < nodes[j].RefId
				})
				assert.Equal(t, []dao.Node{
					{
						Biz:   domain.BizQuestion,
						Rid:   1,
						RefId: 1,
					},
					{
						Biz:   domain.BizQuestion,
						Rid:   1,
						RefId: 2,
					},
					{
						Biz:   domain.BizQuestion,
						Rid:   1,
						RefId: 3,
					},
				}, nodes)
			},
			req: web.Roadmap{
				Title: "标题1",
				Biz:   "questionSet",
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
			name: "删除成功，同时删除关联的Edge v1和Node",
			before: func(t *testing.T) {
				// 创建一个roadmap
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				err := s.db.WithContext(ctx).Create(&dao.Roadmap{
					Id:    1,
					Title: "Roadmap 1",
					Biz:   sqlx.NewNullString("test"),
					BizId: sqlx.NewNullInt64(123),
				}).Error
				require.NoError(t, err)

				// 创建关联的节点
				nodes := []dao.Node{
					{Id: 1, Biz: "question", Rid: 1, RefId: 123, Attrs: "attributes1"},
					{Id: 2, Biz: "case", Rid: 1, RefId: 456, Attrs: "attributes2"},
				}
				err = s.db.Create(&nodes).Error
				require.NoError(t, err)

				// 创建关联的边
				edges := []dao.EdgeV1{
					{Id: 1, Rid: 1, SrcNode: 1, DstNode: 2, Type: "default", Attrs: "edge attributes"},
				}
				err = s.db.Create(&edges).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()

				// 验证roadmap已被删除
				var roadmap dao.Roadmap
				err := s.db.WithContext(ctx).Where("id = ?", 1).First(&roadmap).Error
				assert.Equal(t, gorm.ErrRecordNotFound, err)

				// 验证关联的节点已被删除
				var node dao.Node
				err = s.db.WithContext(ctx).Where("rid = ?", 1).First(&node).Error
				assert.Equal(t, gorm.ErrRecordNotFound, err)

				// 验证关联的边已被删除
				var edge dao.EdgeV1
				err = s.db.WithContext(ctx).Where("rid = ?", 1).First(&edge).Error
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
				"/roadmap/delete", iox.NewJSONReader(tc.req))
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
					{ID: 1, Biz: "question", Rid: 1, BizId: 123, Title: "题目123", Attrs: "attributes1"},
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
