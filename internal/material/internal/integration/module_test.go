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
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/ecodeclub/ekit/iox"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/material/internal/domain"
	"github.com/ecodeclub/webook/internal/material/internal/event"
	evtmocks "github.com/ecodeclub/webook/internal/material/internal/event/mocks"
	"github.com/ecodeclub/webook/internal/material/internal/repository"
	"github.com/ecodeclub/webook/internal/material/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/material/internal/service"
	"github.com/ecodeclub/webook/internal/material/internal/web"
	"github.com/ecodeclub/webook/internal/sms/client"
	smsmocks "github.com/ecodeclub/webook/internal/sms/client/mocks"
	"github.com/ecodeclub/webook/internal/test"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/ecodeclub/webook/internal/user"
	usermocks "github.com/ecodeclub/webook/internal/user/mocks"
	"github.com/ego-component/egorm"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/econf"
	"github.com/gotomicro/ego/server/egin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const testID = int64(223992)

func TestMaterialModule(t *testing.T) {
	suite.Run(t, new(MaterialModuleTestSuite))
}

type MaterialModuleTestSuite struct {
	suite.Suite
	db  *egorm.Component
	svc service.MaterialService
}

func (s *MaterialModuleTestSuite) SetupSuite() {
	s.db = testioc.InitDB()
	s.NoError(dao.InitTables(s.db))
	s.svc = service.NewMaterialService(repository.NewMaterialRepository(dao.NewGORMMaterialDAO(s.db)))
}

func (s *MaterialModuleTestSuite) newGinServer(handler *web.Handler) *egin.Component {
	econf.Set("server", map[string]any{"contextTimeout": "1s"})
	server := egin.Load("server").Build()
	server.Use(func(ctx *gin.Context) {
		ctx.Set("_session", session.NewMemorySession(session.Claims{
			Uid: testID,
		}))
	})

	handler.PrivateRoutes(server.Engine)
	return server
}

func (s *MaterialModuleTestSuite) newAdminGinServer(handler *web.AdminHandler) *egin.Component {
	econf.Set("server", map[string]any{"contextTimeout": "1s"})
	server := egin.Load("server").Build()
	server.Use(func(ctx *gin.Context) {
		ctx.Set("_session", session.NewMemorySession(session.Claims{
			Uid: testID,
		}))
	})

	handler.PrivateRoutes(server.Engine)
	return server
}

func (s *MaterialModuleTestSuite) TearDownSuite() {
	err := s.db.Exec("TRUNCATE TABLE `materials`").Error
	s.NoError(err)
}

func (s *MaterialModuleTestSuite) TestHandler_Submit() {
	t := s.T()

	testCases := []struct {
		name           string
		newHandlerFunc func(t *testing.T, ctrl *gomock.Controller) *web.Handler
		req            web.SubmitMaterialReq

		wantCode int
		wantResp test.Result[any]
		after    func(t *testing.T, req web.SubmitMaterialReq)
	}{
		{
			name: "提交素材成功",
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) *web.Handler {
				t.Helper()
				return web.NewHandler(s.svc)
			},
			req: web.SubmitMaterialReq{
				Material: web.Material{
					AudioURL:  fmt.Sprintf("/%d/audio", testID),
					ResumeURL: fmt.Sprintf("/%d/resume", testID),
					Remark:    "备注内容",
				},
			},
			wantCode: 200,
			wantResp: test.Result[any]{
				Msg: "OK",
			},
			after: func(t *testing.T, req web.SubmitMaterialReq) {
				t.Helper()
				var material domain.Material
				assert.NoError(t, s.db.WithContext(t.Context()).Where("uid = ?", testID).First(&material).Error)
				assert.NotZero(t, material.ID)
				assert.Equal(t, testID, material.Uid)
				assert.Equal(t, req.Material.AudioURL, material.AudioURL)
				assert.Equal(t, req.Material.ResumeURL, material.ResumeURL)
				assert.Equal(t, req.Material.Remark, material.Remark)
				assert.Equal(t, domain.MaterialStatusInit, material.Status)
				assert.NotZero(t, material.Ctime)
				assert.NotZero(t, material.Utime)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			req, err := http.NewRequest(http.MethodPost,
				"/material/submit", iox.NewJSONReader(tc.req))
			require.NoError(t, err)
			req.Header.Set("content-type", "application/json")
			recorder := test.NewJSONResponseRecorder[any]()
			server := s.newGinServer(tc.newHandlerFunc(t, ctrl))
			server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp.Data, recorder.MustScan().Data)
		})
	}
}

func (s *MaterialModuleTestSuite) TestAdminHandler_List() {
	t := s.T()

	err := s.db.Exec("TRUNCATE TABLE `materials`").Error
	require.NoError(t, err)

	total := 10
	for idx := 0; idx < total; idx++ {
		id := int64(3000 + idx)
		_, err := s.svc.Submit(context.Background(), domain.Material{
			Uid:       id,
			AudioURL:  fmt.Sprintf("/%d/admin/audio", id),
			ResumeURL: fmt.Sprintf("/%d/admin/resume", id),
			Remark:    fmt.Sprintf("admin/remark-%d", id),
		})
		require.NoError(t, err)
	}

	listReq := web.ListMaterialsReq{
		Limit:  2,
		Offset: 0,
	}

	req, err := http.NewRequest(http.MethodPost,
		"/material/list", iox.NewJSONReader(listReq))
	require.NoError(t, err)
	req.Header.Set("content-type", "application/json")
	recorder := test.NewJSONResponseRecorder[web.ListMaterialsResp]()
	server := s.newAdminGinServer(web.NewAdminHandler(s.svc, nil, nil, nil))
	server.ServeHTTP(recorder, req)
	require.Equal(t, 200, recorder.Code)
	result := recorder.MustScan()
	require.Equal(t, int64(total), result.Data.Total)
	require.Len(t, result.Data.Materials, listReq.Limit)
}

func (s *MaterialModuleTestSuite) TestAdminHandler_Accept() {
	t := s.T()
	testCases := []struct {
		name           string
		before         func(t *testing.T) int64
		newHandlerFunc func(t *testing.T, ctrl *gomock.Controller, id int64) *web.AdminHandler
		req            web.AcceptMaterialReq

		wantCode int
		wantResp test.Result[any]
		after    func(t *testing.T, id int64)
	}{
		{
			name: "接受素材成功",
			before: func(t *testing.T) int64 {
				t.Helper()
				id, err := s.svc.Submit(t.Context(), domain.Material{
					Uid:       testID,
					AudioURL:  fmt.Sprintf("/%d/admin/audio", testID),
					ResumeURL: fmt.Sprintf("/%d/admin/resume", testID),
					Remark:    fmt.Sprintf("admin/remark-%d", testID),
				})
				require.NoError(t, err)
				return id
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller, id int64) *web.AdminHandler {
				t.Helper()
				producer := evtmocks.NewMockMemberEventProducer(ctrl)
				producer.EXPECT().Produce(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, event event.MemberEvent) error {
					assert.NotZero(t, event.Key)
					assert.Equal(t, testID, event.Uid)
					assert.Equal(t, uint64(30), event.Days)
					assert.Equal(t, "material", event.Biz)
					assert.Equal(t, id, event.BizId)
					assert.Equal(t, "素材被采纳", event.Action)
					return nil
				}).Times(1)
				return web.NewAdminHandler(s.svc, nil, producer, nil)
			},
			req: web.AcceptMaterialReq{
				ID: 0,
			},
			wantCode: 200,
			wantResp: test.Result[any]{
				Msg: "OK",
			},
			after: func(t *testing.T, id int64) {
				t.Helper()
				var material domain.Material
				assert.NoError(t, s.db.WithContext(t.Context()).Where("id = ?", id).First(&material).Error)
				assert.Equal(t, testID, material.Uid)
				assert.Equal(t, fmt.Sprintf("/%d/admin/audio", testID), material.AudioURL)
				assert.Equal(t, fmt.Sprintf("/%d/admin/resume", testID), material.ResumeURL)
				assert.Equal(t, fmt.Sprintf("admin/remark-%d", testID), material.Remark)
				assert.Equal(t, domain.MaterialStatusAccepted, material.Status)
				assert.NotZero(t, material.Ctime)
				assert.NotZero(t, material.Utime)
			},
		},
		{
			name: "接受素材失败_素材ID不存在",
			before: func(t *testing.T) int64 {
				t.Helper()
				return -1
			},
			newHandlerFunc: func(t *testing.T, _ *gomock.Controller, _ int64) *web.AdminHandler {
				t.Helper()
				return web.NewAdminHandler(s.svc, nil, nil, nil)
			},
			req: web.AcceptMaterialReq{
				ID: 0,
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: 518001, Msg: "系统错误",
			},
			after: func(t *testing.T, id int64) {
				t.Helper()
			},
		},
		{
			name: "接受素材失败_福利发放失败",
			before: func(t *testing.T) int64 {
				t.Helper()
				id, err := s.svc.Submit(t.Context(), domain.Material{
					Uid:       testID,
					AudioURL:  fmt.Sprintf("/%d/admin/2/audio", testID),
					ResumeURL: fmt.Sprintf("/%d/admin/2/resume", testID),
					Remark:    fmt.Sprintf("admin/2/remark-%d", testID),
				})
				require.NoError(t, err)
				return id
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller, _ int64) *web.AdminHandler {
				t.Helper()
				producer := evtmocks.NewMockMemberEventProducer(ctrl)
				producer.EXPECT().Produce(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ event.MemberEvent) error {
					return errors.New("fake error")
				}).Times(1)
				return web.NewAdminHandler(s.svc, nil, producer, nil)
			},
			req: web.AcceptMaterialReq{
				ID: 0,
			},
			wantCode: 200,
			wantResp: test.Result[any]{
				Msg: "OK",
			},
			after: func(t *testing.T, id int64) {
				t.Helper()
				var material domain.Material
				assert.NoError(t, s.db.WithContext(t.Context()).Where("id = ?", id).First(&material).Error)
				assert.Equal(t, testID, material.Uid)
				assert.Equal(t, fmt.Sprintf("/%d/admin/2/audio", testID), material.AudioURL)
				assert.Equal(t, fmt.Sprintf("/%d/admin/2/resume", testID), material.ResumeURL)
				assert.Equal(t, fmt.Sprintf("admin/2/remark-%d", testID), material.Remark)
				assert.Equal(t, domain.MaterialStatusAccepted, material.Status)
				assert.NotZero(t, material.Ctime)
				assert.NotZero(t, material.Utime)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			id := tc.before(t)
			tc.req.ID = id

			req, err := http.NewRequest(http.MethodPost,
				"/material/accept", iox.NewJSONReader(tc.req))
			require.NoError(t, err)
			req.Header.Set("content-type", "application/json")
			recorder := test.NewJSONResponseRecorder[any]()
			server := s.newAdminGinServer(tc.newHandlerFunc(t, ctrl, id))
			server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp.Data, recorder.MustScan().Data)

			tc.after(t, id)
		})
	}
}

func (s *MaterialModuleTestSuite) TestAdminHandler_Notify() {
	t := s.T()
	testCases := []struct {
		name           string
		before         func(t *testing.T) int64
		newHandlerFunc func(t *testing.T, ctrl *gomock.Controller) *web.AdminHandler
		req            web.NotifyUserReq

		wantCode int
		wantResp test.Result[any]
		after    func(t *testing.T, id int64)
	}{
		{
			name: "通知用户成功",
			before: func(t *testing.T) int64 {
				t.Helper()
				id, err := s.svc.Submit(t.Context(), domain.Material{
					Uid:       testID,
					AudioURL:  fmt.Sprintf("/%d/admin/3/audio", testID),
					ResumeURL: fmt.Sprintf("/%d/admin/3/resume", testID),
					Remark:    fmt.Sprintf("admin/3/remark-%d", testID),
				})
				require.NoError(t, err)
				err = s.svc.Accept(t.Context(), id)
				require.NoError(t, err)
				return id
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) *web.AdminHandler {
				t.Helper()
				userSvc := usermocks.NewMockUserService(ctrl)
				userSvc.EXPECT().Profile(gomock.Any(), testID).Return(user.User{Id: testID, Phone: "13845016319"}, nil).Times(1)

				cli := smsmocks.NewMockClient(ctrl)
				cli.EXPECT().Send(gomock.Any()).DoAndReturn(func(req client.SendReq) (client.SendResp, error) {
					assert.Contains(t, req.PhoneNumbers, "13845016319")
					assert.NotZero(t, req.TemplateID)
					assert.Equal(t, "2025-7-01 20:00", req.TemplateParam["date"])
					return client.SendResp{
						RequestID: fmt.Sprintf("%d", time.Now().UnixMilli()),
						PhoneNumbers: map[string]client.SendRespStatus{
							"13845016319": {
								Code:    client.OK,
								Message: "发送成功",
							},
						},
					}, nil
				})
				return web.NewAdminHandler(s.svc, userSvc, nil, cli)
			},
			req: web.NotifyUserReq{
				Date: "2025-7-01 20:00",
			},
			wantCode: 200,
			wantResp: test.Result[any]{
				Msg: "OK",
			},
			after: func(t *testing.T, id int64) {
				t.Helper()
				var material domain.Material
				assert.NoError(t, s.db.WithContext(t.Context()).Where("id = ?", id).First(&material).Error)
				assert.Equal(t, testID, material.Uid)
				assert.Equal(t, fmt.Sprintf("/%d/admin/3/audio", testID), material.AudioURL)
				assert.Equal(t, fmt.Sprintf("/%d/admin/3/resume", testID), material.ResumeURL)
				assert.Equal(t, fmt.Sprintf("admin/3/remark-%d", testID), material.Remark)
				assert.Equal(t, domain.MaterialStatusAccepted, material.Status)
				assert.NotZero(t, material.Ctime)
				assert.NotZero(t, material.Utime)
			},
		},
		{
			name: "通知用户失败_素材未被接受",
			before: func(t *testing.T) int64 {
				t.Helper()
				id, err := s.svc.Submit(t.Context(), domain.Material{
					Uid:       testID,
					AudioURL:  fmt.Sprintf("/%d/admin/4/audio", testID),
					ResumeURL: fmt.Sprintf("/%d/admin/4/resume", testID),
					Remark:    fmt.Sprintf("admin/4/remark-%d", testID),
				})
				require.NoError(t, err)
				return id
			},
			newHandlerFunc: func(t *testing.T, _ *gomock.Controller) *web.AdminHandler {
				t.Helper()
				return web.NewAdminHandler(s.svc, nil, nil, nil)
			},
			req: web.NotifyUserReq{
				Date: "2025-7-02 20:00",
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: 518001, Msg: "系统错误",
			},
			after: func(t *testing.T, id int64) {
				t.Helper()
				var material domain.Material
				assert.NoError(t, s.db.WithContext(t.Context()).Where("id = ?", id).First(&material).Error)
				assert.Equal(t, testID, material.Uid)
				assert.Equal(t, fmt.Sprintf("/%d/admin/4/audio", testID), material.AudioURL)
				assert.Equal(t, fmt.Sprintf("/%d/admin/4/resume", testID), material.ResumeURL)
				assert.Equal(t, fmt.Sprintf("admin/4/remark-%d", testID), material.Remark)
				assert.Equal(t, domain.MaterialStatusInit, material.Status)
				assert.NotZero(t, material.Ctime)
				assert.NotZero(t, material.Utime)
			},
		},
		{
			name: "通知用户失败_用户ID不存在",
			before: func(t *testing.T) int64 {
				t.Helper()
				id, err := s.svc.Submit(t.Context(), domain.Material{
					Uid:       testID,
					AudioURL:  fmt.Sprintf("/%d/admin/5/audio", testID),
					ResumeURL: fmt.Sprintf("/%d/admin/5/resume", testID),
					Remark:    fmt.Sprintf("admin/5/remark-%d", testID),
				})
				require.NoError(t, err)
				err = s.svc.Accept(t.Context(), id)
				require.NoError(t, err)
				return id
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) *web.AdminHandler {
				t.Helper()
				userSvc := usermocks.NewMockUserService(ctrl)
				userSvc.EXPECT().Profile(gomock.Any(), testID).Return(user.User{}, errors.New("fake error")).Times(1)
				return web.NewAdminHandler(s.svc, userSvc, nil, nil)
			},
			req: web.NotifyUserReq{
				Date: "2025-7-03 20:00",
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: 518001, Msg: "系统错误",
			},
			after: func(t *testing.T, id int64) {
				t.Helper()
				var material domain.Material
				assert.NoError(t, s.db.WithContext(t.Context()).Where("id = ?", id).First(&material).Error)
				assert.Equal(t, testID, material.Uid)
				assert.Equal(t, fmt.Sprintf("/%d/admin/5/audio", testID), material.AudioURL)
				assert.Equal(t, fmt.Sprintf("/%d/admin/5/resume", testID), material.ResumeURL)
				assert.Equal(t, fmt.Sprintf("admin/5/remark-%d", testID), material.Remark)
				assert.Equal(t, domain.MaterialStatusAccepted, material.Status)
				assert.NotZero(t, material.Ctime)
				assert.NotZero(t, material.Utime)
			},
		},
		{
			name: "通知用户失败_用户未绑定手机号",
			before: func(t *testing.T) int64 {
				t.Helper()
				id, err := s.svc.Submit(t.Context(), domain.Material{
					Uid:       testID,
					AudioURL:  fmt.Sprintf("/%d/admin/6/audio", testID),
					ResumeURL: fmt.Sprintf("/%d/admin/6/resume", testID),
					Remark:    fmt.Sprintf("admin/6/remark-%d", testID),
				})
				require.NoError(t, err)
				err = s.svc.Accept(t.Context(), id)
				require.NoError(t, err)
				return id
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) *web.AdminHandler {
				t.Helper()
				userSvc := usermocks.NewMockUserService(ctrl)
				userSvc.EXPECT().Profile(gomock.Any(), testID).Return(user.User{Id: testID, Phone: ""}, nil).Times(1)
				return web.NewAdminHandler(s.svc, userSvc, nil, nil)
			},
			req: web.NotifyUserReq{
				Date: "2025-7-04 20:00",
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: 418001, Msg: "用户未绑定手机号",
			},
			after: func(t *testing.T, id int64) {
				t.Helper()
				var material domain.Material
				assert.NoError(t, s.db.WithContext(t.Context()).Where("id = ?", id).First(&material).Error)
				assert.Equal(t, testID, material.Uid)
				assert.Equal(t, fmt.Sprintf("/%d/admin/6/audio", testID), material.AudioURL)
				assert.Equal(t, fmt.Sprintf("/%d/admin/6/resume", testID), material.ResumeURL)
				assert.Equal(t, fmt.Sprintf("admin/6/remark-%d", testID), material.Remark)
				assert.Equal(t, domain.MaterialStatusAccepted, material.Status)
				assert.NotZero(t, material.Ctime)
				assert.NotZero(t, material.Utime)
			},
		},
		{
			name: "通知用户失败_发送短信失败",
			before: func(t *testing.T) int64 {
				t.Helper()
				id, err := s.svc.Submit(t.Context(), domain.Material{
					Uid:       testID,
					AudioURL:  fmt.Sprintf("/%d/admin/7/audio", testID),
					ResumeURL: fmt.Sprintf("/%d/admin/7/resume", testID),
					Remark:    fmt.Sprintf("admin/7/remark-%d", testID),
				})
				require.NoError(t, err)
				err = s.svc.Accept(t.Context(), id)
				require.NoError(t, err)
				return id
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) *web.AdminHandler {
				t.Helper()
				t.Helper()
				userSvc := usermocks.NewMockUserService(ctrl)
				userSvc.EXPECT().Profile(gomock.Any(), testID).Return(user.User{Id: testID, Phone: "13845016319"}, nil).Times(1)

				cli := smsmocks.NewMockClient(ctrl)
				cli.EXPECT().Send(gomock.Any()).DoAndReturn(func(req client.SendReq) (client.SendResp, error) {
					assert.Contains(t, req.PhoneNumbers, "13845016319")
					assert.NotZero(t, req.TemplateID)
					assert.Equal(t, "2025-7-05 20:00", req.TemplateParam["date"])
					return client.SendResp{}, errors.New("fake error")
				})
				return web.NewAdminHandler(s.svc, userSvc, nil, cli)
			},
			req: web.NotifyUserReq{
				Date: "2025-7-05 20:00",
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: 518001, Msg: "系统错误",
			},
			after: func(t *testing.T, id int64) {
				t.Helper()
				var material domain.Material
				assert.NoError(t, s.db.WithContext(t.Context()).Where("id = ?", id).First(&material).Error)
				assert.Equal(t, testID, material.Uid)
				assert.Equal(t, fmt.Sprintf("/%d/admin/7/audio", testID), material.AudioURL)
				assert.Equal(t, fmt.Sprintf("/%d/admin/7/resume", testID), material.ResumeURL)
				assert.Equal(t, fmt.Sprintf("admin/7/remark-%d", testID), material.Remark)
				assert.Equal(t, domain.MaterialStatusAccepted, material.Status)
				assert.NotZero(t, material.Ctime)
				assert.NotZero(t, material.Utime)
			},
		},
		{
			name: "通知用户失败_用户接收失败",
			before: func(t *testing.T) int64 {
				t.Helper()
				id, err := s.svc.Submit(t.Context(), domain.Material{
					Uid:       testID,
					AudioURL:  fmt.Sprintf("/%d/admin/8/audio", testID),
					ResumeURL: fmt.Sprintf("/%d/admin/8/resume", testID),
					Remark:    fmt.Sprintf("admin/8/remark-%d", testID),
				})
				require.NoError(t, err)
				err = s.svc.Accept(t.Context(), id)
				require.NoError(t, err)
				return id
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) *web.AdminHandler {
				t.Helper()
				userSvc := usermocks.NewMockUserService(ctrl)
				userSvc.EXPECT().Profile(gomock.Any(), testID).Return(user.User{Id: testID, Phone: "13845016329"}, nil).Times(1)

				cli := smsmocks.NewMockClient(ctrl)
				cli.EXPECT().Send(gomock.Any()).DoAndReturn(func(req client.SendReq) (client.SendResp, error) {
					assert.Contains(t, req.PhoneNumbers, "13845016329")
					assert.NotZero(t, req.TemplateID)
					assert.Equal(t, "2025-7-06 20:00", req.TemplateParam["date"])
					return client.SendResp{
						RequestID: fmt.Sprintf("%d", time.Now().UnixMilli()),
						PhoneNumbers: map[string]client.SendRespStatus{
							"13845016329": {
								Code:    "Failed",
								Message: "用户已停机",
							},
						},
					}, nil
				})
				return web.NewAdminHandler(s.svc, userSvc, nil, cli)
			},
			req: web.NotifyUserReq{
				Date: "2025-7-06 20:00",
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: 518002, Msg: "用户接收通知失败",
			},
			after: func(t *testing.T, id int64) {
				t.Helper()
				var material domain.Material
				assert.NoError(t, s.db.WithContext(t.Context()).Where("id = ?", id).First(&material).Error)
				assert.Equal(t, testID, material.Uid)
				assert.Equal(t, fmt.Sprintf("/%d/admin/8/audio", testID), material.AudioURL)
				assert.Equal(t, fmt.Sprintf("/%d/admin/8/resume", testID), material.ResumeURL)
				assert.Equal(t, fmt.Sprintf("admin/8/remark-%d", testID), material.Remark)
				assert.Equal(t, domain.MaterialStatusAccepted, material.Status)
				assert.NotZero(t, material.Ctime)
				assert.NotZero(t, material.Utime)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			id := tc.before(t)
			tc.req.ID = id

			req, err := http.NewRequest(http.MethodPost,
				"/material/notify", iox.NewJSONReader(tc.req))
			require.NoError(t, err)
			req.Header.Set("content-type", "application/json")
			recorder := test.NewJSONResponseRecorder[any]()
			server := s.newAdminGinServer(tc.newHandlerFunc(t, ctrl))
			server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp.Data, recorder.MustScan().Data)

			tc.after(t, id)
		})
	}
}
