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
	"errors"
	"net/http"
	"testing"

	"github.com/ecodeclub/webook/internal/kbase/internal/domain"
	"github.com/ecodeclub/webook/internal/kbase/internal/integration/startup"
	"github.com/ecodeclub/webook/internal/kbase/internal/web"
	kbasemocks "github.com/ecodeclub/webook/internal/kbase/mocks"
	"github.com/ecodeclub/webook/internal/roadmap"
	roadmapmocks "github.com/ecodeclub/webook/internal/roadmap/mocks"

	"github.com/ecodeclub/ekit/iox"
	baguwen "github.com/ecodeclub/webook/internal/question"
	quemocks "github.com/ecodeclub/webook/internal/question/mocks"
	"github.com/ecodeclub/webook/internal/test"
	"github.com/ego-component/egorm"
	"github.com/gotomicro/ego/core/econf"
	"github.com/gotomicro/ego/server/egin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestAdminHandler(t *testing.T) {
	suite.Run(t, new(AdminHandlerTestSuite))
}

type AdminHandlerTestSuite struct {
	suite.Suite
	db         *egorm.Component
	server     *egin.Component
	hdl        *web.AdminHandler
	ctrl       *gomock.Controller
	mockQueSvc *quemocks.MockService
	mockRdSvc  *roadmapmocks.MockAdminService
	mockSvc    *kbasemocks.MockService
}

func (s *AdminHandlerTestSuite) SetupSuite() {
	s.ctrl = gomock.NewController(s.T())
	s.mockQueSvc = quemocks.NewMockService(s.ctrl)
	s.mockRdSvc = roadmapmocks.NewMockAdminService(s.ctrl)
	s.mockSvc = kbasemocks.NewMockService(s.ctrl)

	m := startup.InitModule(&baguwen.Module{
		Svc: s.mockQueSvc,
	}, &roadmap.Module{
		AdminSvc: s.mockRdSvc,
	}, s.mockSvc)
	s.hdl = m.AdminHdl

	econf.Set("server", map[string]any{"contextTimeout": "10s"})
	server := egin.Load("server").Build()
	s.hdl.PrivateRoutes(server.Engine)
	s.server = server
}

func (s *AdminHandlerTestSuite) TearDownTest() {
	if s.db != nil {
		err := s.db.Exec("TRUNCATE TABLE roadmaps").Error
		require.NoError(s.T(), err)
		err = s.db.Exec("TRUNCATE TABLE roadmap_edges").Error
		require.NoError(s.T(), err)
		err = s.db.Exec("TRUNCATE TABLE roadmap_nodes").Error
		require.NoError(s.T(), err)
		err = s.db.Exec("TRUNCATE TABLE roadmap_edges_v1").Error
		require.NoError(s.T(), err)
	}
}

func (s *AdminHandlerTestSuite) TestUpsert() {
	t := s.T()
	testCases := []struct {
		name     string
		req      web.Req
		setup    func()
		wantCode int
		wantResp test.Result[any]
	}{
		{
			name: "成功-同步question",
			req: web.Req{
				Biz:   domain.BizQuestion,
				BizID: 123,
			},
			setup: func() {
				s.mockQueSvc.EXPECT().PubDetailWithoutCntView(gomock.Any(), int64(123)).
					Return(baguwen.Question{
						Id:    123,
						Title: "题目123",
					}, nil).Times(1)
				s.mockSvc.EXPECT().BulkUpsert(gomock.Any(), "question_index", gomock.Any()).
					Return(nil).Times(1)
			},
			wantCode: 200,
			wantResp: test.Result[any]{
				Msg: "ok",
			},
		},
		{
			name: "失败-未知业务类型",
			req: web.Req{
				Biz:   "unknown_biz",
				BizID: 123,
			},
			setup:    func() {},
			wantCode: 200,
			wantResp: test.Result[any]{
				Code: 520001,
				Msg:  "系统错误",
			},
		},
		{
			name: "失败-service返回错误",
			req: web.Req{
				Biz:   domain.BizQuestion,
				BizID: 123,
			},
			setup: func() {
				s.mockQueSvc.EXPECT().PubDetailWithoutCntView(gomock.Any(), int64(123)).
					Return(baguwen.Question{
						Id:    123,
						Title: "题目123",
					}, nil).Times(1)
				s.mockSvc.EXPECT().BulkUpsert(gomock.Any(), "question_index", gomock.Any()).
					Return(errors.New("ES错误")).Times(1)
			},
			wantCode: 200,
			wantResp: test.Result[any]{
				Code: 520001,
				Msg:  "系统错误",
			},
		},
		{
			name: "成功-同步question_rel",
			req: web.Req{
				Biz:   domain.BizQuestionRel,
				BizID: 123,
			},
			setup: func() {
				s.mockRdSvc.EXPECT().Detail(gomock.Any(), int64(123)).
					Return(roadmap.Roadmap{
						Id: 123,
						Biz: roadmap.Biz{
							Biz:   "questionSet",
							BizId: 456,
						},
						Edges: []roadmap.Edge{
							{
								Id:  1,
								Src: roadmap.Node{ID: 10, Rid: 123, Biz: roadmap.Biz{Biz: "question", BizId: 10}},
								Dst: roadmap.Node{ID: 20, Rid: 123, Biz: roadmap.Biz{Biz: "question", BizId: 20}},
							},
						},
					}, nil).Times(1)
				s.mockSvc.EXPECT().BulkUpsert(gomock.Any(), "question_rel_index", gomock.Any()).
					Return(nil).Times(1)
			},
			wantCode: 200,
			wantResp: test.Result[any]{
				Msg: "ok",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			req, err := http.NewRequestWithContext(t.Context(), http.MethodPost,
				"/kbase/sync/upsert", iox.NewJSONReader(tc.req))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			recorder := test.NewJSONResponseRecorder[any]()
			s.server.ServeHTTP(recorder, req)

			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
		})
	}
}

func (s *AdminHandlerTestSuite) TestBatchUpsert() {
	t := s.T()
	testCases := []struct {
		name     string
		req      web.BatchUpsertReq
		setup    func()
		wantCode int
		wantResp test.Result[any]
	}{
		{
			name: "成功-批量同步question",
			req: web.BatchUpsertReq{
				Biz:   domain.BizQuestion,
				Since: 1000,
			},
			setup: func() {
				s.mockQueSvc.EXPECT().ListPubSince(gomock.Any(), int64(1000), 0, 100).
					Return([]baguwen.Question{}, nil).Times(1)
			},
			wantCode: 200,
			wantResp: test.Result[any]{
				Msg: "ok",
			},
		},
		{
			name: "失败-未知业务类型",
			req: web.BatchUpsertReq{
				Biz:   "unknown_biz",
				Since: 1000,
			},
			setup:    func() {},
			wantCode: 200,
			wantResp: test.Result[any]{
				Code: 520001,
				Msg:  "系统错误",
			},
		},
		{
			name: "失败-service返回错误",
			req: web.BatchUpsertReq{
				Biz:   domain.BizQuestion,
				Since: 1000,
			},
			setup: func() {
				s.mockQueSvc.EXPECT().ListPubSince(gomock.Any(), int64(1000), 0, 100).
					Return(nil, errors.New("查询失败")).Times(1)
			},
			wantCode: 200,
			wantResp: test.Result[any]{
				Code: 520001,
				Msg:  "系统错误",
			},
		},
		{
			name: "成功-批量同步question_rel",
			req: web.BatchUpsertReq{
				Biz:   domain.BizQuestionRel,
				Since: 1000,
			},
			setup: func() {
				s.mockRdSvc.EXPECT().ListSince(gomock.Any(), int64(1000), 0, 100).
					Return([]roadmap.Roadmap{}, nil).Times(1)
			},
			wantCode: 200,
			wantResp: test.Result[any]{
				Msg: "ok",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			req, err := http.NewRequestWithContext(t.Context(), http.MethodPost,
				"/kbase/sync/batch-upsert", iox.NewJSONReader(tc.req))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			recorder := test.NewJSONResponseRecorder[any]()
			s.server.ServeHTTP(recorder, req)

			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
		})
	}
}

func (s *AdminHandlerTestSuite) TestDelete() {
	t := s.T()
	testCases := []struct {
		name     string
		req      web.Req
		setup    func()
		wantCode int
		wantResp test.Result[any]
	}{
		{
			name: "成功-删除question",
			req: web.Req{
				Biz:   domain.BizQuestion,
				BizID: 123,
			},
			setup: func() {
				s.mockSvc.EXPECT().BulkDelete(gomock.Any(), "question_index", []string{"123"}).
					Return(nil).Times(1)
			},
			wantCode: 200,
			wantResp: test.Result[any]{
				Msg: "ok",
			},
		},
		{
			name: "失败-未知业务类型",
			req: web.Req{
				Biz:   "unknown_biz",
				BizID: 123,
			},
			setup:    func() {},
			wantCode: 200,
			wantResp: test.Result[any]{
				Code: 520001,
				Msg:  "系统错误",
			},
		},
		{
			name: "失败-service返回错误",
			req: web.Req{
				Biz:   domain.BizQuestion,
				BizID: 123,
			},
			setup: func() {
				s.mockSvc.EXPECT().BulkDelete(gomock.Any(), "question_index", []string{"123"}).
					Return(errors.New("ES错误")).Times(1)
			},
			wantCode: 200,
			wantResp: test.Result[any]{
				Code: 520001,
				Msg:  "系统错误",
			},
		},
		{
			name: "成功-删除question_rel",
			req: web.Req{
				Biz:   domain.BizQuestionRel,
				BizID: 123,
			},
			setup: func() {
				s.mockSvc.EXPECT().BulkDelete(gomock.Any(), "question_rel_index", []string{"123"}).
					Return(nil).Times(1)
			},
			wantCode: 200,
			wantResp: test.Result[any]{
				Msg: "ok",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			req, err := http.NewRequestWithContext(t.Context(), http.MethodPost,
				"/kbase/sync/delete", iox.NewJSONReader(tc.req))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			recorder := test.NewJSONResponseRecorder[any]()
			s.server.ServeHTTP(recorder, req)

			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
		})
	}
}
