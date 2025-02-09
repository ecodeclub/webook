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

	"github.com/ecodeclub/ginx/session"

	"gorm.io/gorm"

	"github.com/ecodeclub/webook/internal/permission"

	"github.com/ecodeclub/webook/internal/interactive"
	intrmocks "github.com/ecodeclub/webook/internal/interactive/mocks"
	"go.uber.org/mock/gomock"

	"github.com/ecodeclub/ekit/iox"
	"github.com/ecodeclub/ekit/sqlx"
	"github.com/ecodeclub/webook/internal/project"
	"github.com/ecodeclub/webook/internal/project/internal/domain"
	"github.com/ecodeclub/webook/internal/project/internal/integration/startup"
	"github.com/ecodeclub/webook/internal/project/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/project/internal/web"
	"github.com/ecodeclub/webook/internal/test"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/ego-component/egorm"
	"github.com/gotomicro/ego/core/econf"
	"github.com/gotomicro/ego/server/egin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type AdminProjectTestSuite struct {
	suite.Suite
	hdl    *project.AdminHandler
	server *egin.Component

	db          *egorm.Component
	adminPrjDAO dao.ProjectAdminDAO
	prjDAO      dao.ProjectDAO
}

func (s *AdminProjectTestSuite) SetupSuite() {
	ctrl := gomock.NewController(s.T())
	intrSvc := intrmocks.NewMockService(ctrl)
	intrModule := &interactive.Module{
		Svc: intrSvc,
	}
	permModule := &permission.Module{}
	m, err := startup.InitModule(intrModule, permModule, session.DefaultProvider())
	require.NoError(s.T(), err)
	s.hdl = m.AdminHdl

	econf.Set("server", map[string]any{"contextTimeout": "10s"})
	server := egin.Load("server").Build()
	s.hdl.PrivateRoutes(server.Engine)
	s.server = server
	s.db = testioc.InitDB()
	s.adminPrjDAO = dao.NewGORMProjectAdminDAO(s.db)
	s.prjDAO = dao.NewGORMProjectDAO(s.db)
}

func (s *AdminProjectTestSuite) TearDownTest() {
	err := s.db.Exec("TRUNCATE TABLE projects;").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE pub_projects;").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE project_difficulties;").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE pub_project_difficulties;").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE project_resumes;").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE pub_project_resumes;").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE project_questions;").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE pub_project_questions;").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE project_introductions;").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE pub_project_introductions;").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE project_combos;").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE pub_project_combos;").Error
	require.NoError(s.T(), err)
}

// TestProjectSave 测试 Project 本身数据的保存
func (s *AdminProjectTestSuite) TestProjectSave() {
	testCases := []struct {
		name   string
		before func(t *testing.T)
		after  func(t *testing.T)

		req      web.Project
		wantCode int
		wantResp test.Result[int64]
	}{
		{
			name: "保存成功-新建",
			before: func(t *testing.T) {

			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				// 验证数据库中有数据
				prj, err := s.adminPrjDAO.GetById(ctx, 1)
				require.NoError(t, err)
				assert.True(t, prj.Ctime > 0)
				prj.Ctime = 0
				assert.True(t, prj.Utime > 0)
				prj.Utime = 0
				assert.NotEmpty(t, prj.SN)
				prj.SN = ""
				assert.Equal(t, dao.Project{
					Id:             1,
					Title:          "项目1",
					Desc:           "这是测试项目1",
					Overview:       "这是测试项目1 overview",
					SystemDesign:   "这是测试项目1 SystemDesign",
					RefQuestionSet: 444,
					GithubRepo:     "github1",
					GiteeRepo:      "gitee1",
					CodeSPU:        sqlx.NewNullString("codeSPU1"),
					ProductSPU:     sqlx.NewNullString("productSPU1"),
					Status:         domain.ProjectStatusUnpublished.ToUint8(),
					Labels: sqlx.JsonColumn[[]string]{
						Val:   []string{"标签1"},
						Valid: true,
					},
				}, prj)
			},
			req: web.Project{
				Title:          "项目1",
				Desc:           "这是测试项目1",
				Overview:       "这是测试项目1 overview",
				SystemDesign:   "这是测试项目1 SystemDesign",
				RefQuestionSet: 444,
				GithubRepo:     "github1",
				GiteeRepo:      "gitee1",
				CodeSPU:        "codeSPU1",
				ProductSPU:     "productSPU1",
				Labels:         []string{"标签1"},
			},
			wantCode: 200,
			wantResp: test.Result[int64]{
				Data: 1,
			},
		},
		{
			name: "保存成功-更新",
			before: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				err := s.db.WithContext(ctx).Create(&dao.Project{
					Id:             12,
					Title:          "老的标题",
					SN:             "old-SN",
					Overview:       "这是老的测试项目1 overview",
					SystemDesign:   "这是老的测试项目1 SystemDesign",
					RefQuestionSet: 555,
					GithubRepo:     "老的 github1",
					GiteeRepo:      "老的 gitee1",
					CodeSPU:        sqlx.NewNullString("老的 codeSPU1"),
					ProductSPU:     sqlx.NewNullString("老的 productSPU1"),
					Status:         domain.ProjectStatusPublished.ToUint8(),
					Labels: sqlx.JsonColumn[[]string]{
						Val:   []string{"标签3", "标签1"},
						Valid: true,
					},
					Desc: "老的描述",
					// ctime 应该不会被更新
					Ctime: 123,
					Utime: 123,
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				// 验证数据库中有数据
				prj, err := s.adminPrjDAO.GetById(ctx, 12)
				require.NoError(t, err)
				assert.True(t, prj.Utime > 123)
				prj.Utime = 0
				assert.Equal(t, dao.Project{
					Id:             12,
					Title:          "项目1",
					SN:             "old-SN",
					Desc:           "这是测试项目1",
					Overview:       "这是新的测试项目1 overview",
					SystemDesign:   "这是新的测试项目1 SystemDesign",
					RefQuestionSet: 444,
					GithubRepo:     "github1",
					GiteeRepo:      "gitee1",
					CodeSPU:        sqlx.NewNullString("codeSPU1"),
					ProductSPU:     sqlx.NewNullString("productSPU1"),
					Status:         domain.ProjectStatusUnpublished.ToUint8(),
					Labels: sqlx.JsonColumn[[]string]{
						Val:   []string{"标签1"},
						Valid: true,
					},
					Ctime: 123,
				}, prj)
			},
			req: web.Project{
				Id:             12,
				Title:          "项目1",
				SN:             "new-SN",
				Desc:           "这是测试项目1",
				RefQuestionSet: 444,
				GithubRepo:     "github1",
				GiteeRepo:      "gitee1",
				CodeSPU:        "codeSPU1",
				ProductSPU:     "productSPU1",
				Overview:       "这是新的测试项目1 overview",
				SystemDesign:   "这是新的测试项目1 SystemDesign",
				Labels:         []string{"标签1"},
			},
			wantCode: 200,
			wantResp: test.Result[int64]{
				Data: 12,
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/project/save", iox.NewJSONReader(tc.req))
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

func (s *AdminProjectTestSuite) TestProjectPublish() {
	testCases := []struct {
		name   string
		before func(t *testing.T)
		after  func(t *testing.T)

		req      web.Project
		wantCode int
		wantResp test.Result[int64]
	}{
		{
			name: "发表成功-新建",
			before: func(t *testing.T) {

			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				// 验证数据库中有数据
				prj, err := s.adminPrjDAO.GetById(ctx, 1)
				require.NoError(t, err)
				assert.True(t, prj.Ctime > 0)
				prj.Ctime = 0
				assert.True(t, prj.Utime > 0)
				prj.Utime = 0
				assert.NotEmpty(t, prj.SN)
				prj.SN = ""
				assert.Equal(t, dao.Project{
					Id:             1,
					Title:          "项目1",
					Desc:           "这是测试项目1",
					Overview:       "这是测试项目1 overview",
					SystemDesign:   "这是测试项目1 SystemDesign",
					CodeSPU:        sqlx.NewNullString("codeSPU1"),
					ProductSPU:     sqlx.NewNullString("productSPU1"),
					RefQuestionSet: 444,
					GithubRepo:     "github1",
					GiteeRepo:      "gitee1",
					Status:         domain.ProjectStatusPublished.ToUint8(),
					Labels: sqlx.JsonColumn[[]string]{
						Val:   []string{"标签1"},
						Valid: true,
					},
				}, prj)

				pubPrj, err := s.prjDAO.GetById(ctx, 1)
				require.NoError(t, err)
				assert.True(t, pubPrj.Ctime > 0)
				pubPrj.Ctime = 0
				assert.True(t, pubPrj.Utime > 0)
				pubPrj.Utime = 0
				assert.NotEmpty(t, pubPrj.SN)
				pubPrj.SN = ""
				assert.Equal(t, dao.PubProject{
					Id:             1,
					Title:          "项目1",
					Desc:           "这是测试项目1",
					Overview:       "这是测试项目1 overview",
					SystemDesign:   "这是测试项目1 SystemDesign",
					RefQuestionSet: 444,
					GithubRepo:     "github1",
					GiteeRepo:      "gitee1",
					CodeSPU:        sqlx.NewNullString("codeSPU1"),
					ProductSPU:     sqlx.NewNullString("productSPU1"),
					Status:         domain.ProjectStatusPublished.ToUint8(),
					Labels: sqlx.JsonColumn[[]string]{
						Val:   []string{"标签1"},
						Valid: true,
					},
				}, pubPrj)
			},
			req: web.Project{
				Title:          "项目1",
				Desc:           "这是测试项目1",
				Overview:       "这是测试项目1 overview",
				SystemDesign:   "这是测试项目1 SystemDesign",
				RefQuestionSet: 444,
				GithubRepo:     "github1",
				GiteeRepo:      "gitee1",
				CodeSPU:        "codeSPU1",
				ProductSPU:     "productSPU1",
				Labels:         []string{"标签1"},
			},
			wantCode: 200,
			wantResp: test.Result[int64]{
				Data: 1,
			},
		},
		{
			name: "发表成功-新建更新混合",
			before: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				err := s.db.WithContext(ctx).Create(&dao.Project{
					Id:             13,
					Title:          "老的标题",
					SN:             "old-SN",
					Overview:       "这是老的测试项目1 overview",
					SystemDesign:   "这是老的测试项目1 SystemDesign",
					RefQuestionSet: 555,
					GithubRepo:     "老的 github1",
					GiteeRepo:      "老的 gitee1",
					CodeSPU:        sqlx.NewNullString("老的 codeSPU1"),
					ProductSPU:     sqlx.NewNullString("老的 productSPU1"),
					Status:         domain.ProjectStatusUnpublished.ToUint8(),
					Labels: sqlx.JsonColumn[[]string]{
						Val:   []string{"标签3", "标签1"},
						Valid: true,
					},
					Desc: "老的描述",
					// ctime 应该不会被更新
					Ctime: 123,
					Utime: 123,
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				// 验证数据库中有数据
				prj, err := s.adminPrjDAO.GetById(ctx, 13)
				require.NoError(t, err)
				assert.True(t, prj.Utime > 123)
				prj.Utime = 0
				assert.Equal(t, dao.Project{
					Id:             13,
					Title:          "项目1",
					SN:             "old-SN",
					Desc:           "这是测试项目1",
					Overview:       "这是测试项目1 overview",
					SystemDesign:   "这是测试项目1 SystemDesign",
					RefQuestionSet: 444,
					GithubRepo:     "github1",
					GiteeRepo:      "gitee1",
					CodeSPU:        sqlx.NewNullString("codeSPU1"),
					ProductSPU:     sqlx.NewNullString("productSPU1"),
					Status:         domain.ProjectStatusPublished.ToUint8(),
					Labels: sqlx.JsonColumn[[]string]{
						Val:   []string{"标签1"},
						Valid: true,
					},
					Ctime: 123,
				}, prj)

				// 验证数据库中有数据
				pubPrj, err := s.prjDAO.GetById(ctx, 13)
				require.NoError(t, err)
				assert.True(t, pubPrj.Ctime > 0)
				pubPrj.Ctime = 0
				assert.True(t, pubPrj.Utime > 123)
				pubPrj.Utime = 0
				assert.Equal(t, dao.PubProject{
					Id:             13,
					Title:          "项目1",
					SN:             "old-SN",
					Desc:           "这是测试项目1",
					Overview:       "这是测试项目1 overview",
					SystemDesign:   "这是测试项目1 SystemDesign",
					RefQuestionSet: 444,
					GithubRepo:     "github1",
					GiteeRepo:      "gitee1",
					CodeSPU:        sqlx.NewNullString("codeSPU1"),
					ProductSPU:     sqlx.NewNullString("productSPU1"),
					Status:         domain.ProjectStatusPublished.ToUint8(),
					Labels: sqlx.JsonColumn[[]string]{
						Val:   []string{"标签1"},
						Valid: true,
					},
				}, pubPrj)
			},
			req: web.Project{
				Id:             13,
				Title:          "项目1",
				SN:             "old-SN",
				Desc:           "这是测试项目1",
				Overview:       "这是测试项目1 overview",
				SystemDesign:   "这是测试项目1 SystemDesign",
				RefQuestionSet: 444,
				GithubRepo:     "github1",
				GiteeRepo:      "gitee1",
				CodeSPU:        "codeSPU1",
				ProductSPU:     "productSPU1",
				Labels:         []string{"标签1"},
			},
			wantCode: 200,
			wantResp: test.Result[int64]{
				Data: 13,
			},
		},
		{
			name: "发表成功-更新",
			before: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				err := s.db.WithContext(ctx).Create(&dao.Project{
					Id:             12,
					Title:          "老的标题",
					SN:             "old-SN",
					Overview:       "这是老的测试项目1 overview",
					SystemDesign:   "这是老的测试项目1 SystemDesign",
					RefQuestionSet: 555,
					GithubRepo:     "老的 github1",
					GiteeRepo:      "老的 gitee1",
					CodeSPU:        sqlx.NewNullString("老的 codeSPU1"),
					ProductSPU:     sqlx.NewNullString("老的 productSPU1"),
					Status:         domain.ProjectStatusUnpublished.ToUint8(),
					Labels: sqlx.JsonColumn[[]string]{
						Val:   []string{"标签3", "标签1"},
						Valid: true,
					},
					Desc: "老的描述",
					// ctime 应该不会被更新
					Ctime: 123,
					Utime: 123,
				}).Error
				require.NoError(t, err)

				err = s.db.WithContext(ctx).Create(&dao.PubProject{
					Id:             12,
					Title:          "老的标题",
					SN:             "old-SN",
					Overview:       "这是老的测试项目1 overview",
					SystemDesign:   "这是老的测试项目1 SystemDesign",
					RefQuestionSet: 555,
					GithubRepo:     "老的 github1",
					GiteeRepo:      "老的 gitee1",
					CodeSPU:        sqlx.NewNullString("老的 codeSPU1"),
					ProductSPU:     sqlx.NewNullString("老的 productSPU1"),
					Status:         domain.ProjectStatusUnpublished.ToUint8(),
					Labels: sqlx.JsonColumn[[]string]{
						Val:   []string{"标签3", "标签1"},
						Valid: true,
					},
					Desc: "老的描述",
					// ctime 应该不会被更新
					Ctime: 123,
					Utime: 123,
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				// 验证数据库中有数据
				prj, err := s.adminPrjDAO.GetById(ctx, 12)
				require.NoError(t, err)
				assert.True(t, prj.Utime > 123)
				prj.Utime = 0
				assert.Equal(t, dao.Project{
					Id:             12,
					Title:          "项目1",
					SN:             "old-SN",
					Desc:           "这是测试项目1",
					Overview:       "这是测试项目1 overview",
					SystemDesign:   "这是测试项目1 SystemDesign",
					RefQuestionSet: 444,
					GithubRepo:     "github1",
					GiteeRepo:      "gitee1",
					CodeSPU:        sqlx.NewNullString("codeSPU1"),
					ProductSPU:     sqlx.NewNullString("productSPU1"),
					Status:         domain.ProjectStatusPublished.ToUint8(),
					Labels: sqlx.JsonColumn[[]string]{
						Val:   []string{"标签1"},
						Valid: true,
					},
					Ctime: 123,
				}, prj)

				// 验证数据库中有数据
				pubPrj, err := s.prjDAO.GetById(ctx, 12)
				require.NoError(t, err)
				assert.True(t, pubPrj.Utime > 123)
				pubPrj.Utime = 0
				assert.Equal(t, dao.PubProject{
					Id:             12,
					Title:          "项目1",
					SN:             "old-SN",
					Desc:           "这是测试项目1",
					Overview:       "这是测试项目1 overview",
					SystemDesign:   "这是测试项目1 SystemDesign",
					RefQuestionSet: 444,
					GithubRepo:     "github1",
					GiteeRepo:      "gitee1",
					CodeSPU:        sqlx.NewNullString("codeSPU1"),
					ProductSPU:     sqlx.NewNullString("productSPU1"),
					Status:         domain.ProjectStatusPublished.ToUint8(),
					Labels: sqlx.JsonColumn[[]string]{
						Val:   []string{"标签1"},
						Valid: true,
					},
					Ctime: 123,
				}, pubPrj)
			},
			req: web.Project{
				Id:             12,
				Title:          "项目1",
				SN:             "new-SN",
				Desc:           "这是测试项目1",
				Overview:       "这是测试项目1 overview",
				SystemDesign:   "这是测试项目1 SystemDesign",
				RefQuestionSet: 444,
				GithubRepo:     "github1",
				GiteeRepo:      "gitee1",
				CodeSPU:        "codeSPU1",
				ProductSPU:     "productSPU1",
				Labels:         []string{"标签1"},
			},
			wantCode: 200,
			wantResp: test.Result[int64]{
				Data: 12,
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/project/publish", iox.NewJSONReader(tc.req))
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

func (s *AdminProjectTestSuite) TestProjectDelete() {
	testCases := []struct {
		name string

		before func(t *testing.T)
		after  func(t *testing.T)

		req      web.IdReq
		wantCode int
		wantResp test.Result[any]
	}{
		{
			name: "删除成功",
			before: func(t *testing.T) {
				err := s.db.Create(&dao.Project{Id: 11}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				var prj dao.Project
				err := s.db.Where("id = ?", 11).First(&prj).Error
				require.Equal(t, gorm.ErrRecordNotFound, err)
			},
			req: web.IdReq{
				Id: 11,
			},
			wantCode: 200,
			wantResp: test.Result[any]{Msg: "OK"},
		},
		{
			name: "id不存在",
			before: func(t *testing.T) {
			},
			after: func(t *testing.T) {
			},
			req: web.IdReq{
				Id: 12,
			},
			wantCode: 200,
			wantResp: test.Result[any]{Msg: "OK"},
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/project/delete", iox.NewJSONReader(tc.req))
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

func (s *AdminProjectTestSuite) TestProjectList() {
	prjs := make([]dao.Project, 0, 10)
	for i := 0; i < 10; i++ {
		prjs = append(prjs, s.mockProject(int64(i+1)))
	}
	err := s.db.Create(&prjs).Error
	require.NoError(s.T(), err)
	testCases := []struct {
		name string
		req  web.Page

		wantCode int
		wantResp test.Result[web.ProjectList]
	}{
		{
			name: "从头获取成功",
			req: web.Page{
				Offset: 0,
				Limit:  2,
			},
			wantCode: 200,
			wantResp: test.Result[web.ProjectList]{
				Data: web.ProjectList{
					Total: 10,
					Projects: []web.Project{
						{
							Id:     10,
							Title:  "标题10",
							SN:     "SN10",
							Status: domain.ProjectStatusUnpublished.ToUint8(),
							Labels: []string{"标签10"},
							Desc:   "描述10",
							Utime:  10,
						},
						{
							Id:     9,
							Title:  "标题9",
							SN:     "SN9",
							Status: domain.ProjectStatusUnpublished.ToUint8(),
							Labels: []string{"标签9"},
							Desc:   "描述9",
							Utime:  9,
						},
					},
				},
			},
		},
		{
			name: "末尾部分获取",
			req: web.Page{
				Offset: 9,
				Limit:  2,
			},
			wantCode: 200,
			wantResp: test.Result[web.ProjectList]{
				Data: web.ProjectList{
					Total: 10,
					Projects: []web.Project{
						{
							Id:     1,
							SN:     "SN1",
							Title:  "标题1",
							Status: domain.ProjectStatusUnpublished.ToUint8(),
							Labels: []string{"标签1"},
							Desc:   "描述1",
							Utime:  1,
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.T().Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost,
				"/project/list", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.ProjectList]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
		})
	}
}

func (s *AdminProjectTestSuite) TestProjectDetail() {
	// 插入各种数据
	prj := s.mockProject(1)
	err := s.db.Create(&prj).Error
	require.NoError(s.T(), err)
	// 难点
	diff := s.mockDiff(1, 1)
	err = s.db.Create(&diff).Error
	require.NoError(s.T(), err)
	// 简历
	rsm := s.mockRsm(1, 1)
	err = s.db.Create(&rsm).Error
	require.NoError(s.T(), err)
	// 项目介绍
	intr := s.mockIntr(1, 1)
	err = s.db.Create(&intr).Error
	require.NoError(s.T(), err)

	que := s.mockQue(1, 1)
	err = s.db.Create(&que).Error
	require.NoError(s.T(), err)

	combo := s.mockCombo(1, 1)
	err = s.db.Create(&combo).Error
	require.NoError(s.T(), err)

	testCases := []struct {
		name string
		req  web.IdReq

		wantCode int
		wantResp test.Result[web.Project]
	}{
		{
			name:     "获取成功",
			req:      web.IdReq{Id: 1},
			wantCode: 200,
			wantResp: test.Result[web.Project]{
				Data: web.Project{
					Id:             1,
					Title:          "标题1",
					SN:             "SN1",
					Overview:       "概览1",
					SystemDesign:   "系统设计1",
					RefQuestionSet: 1,
					GithubRepo:     "github1",
					GiteeRepo:      "gitee1",
					ProductSPU:     "productSPU1",
					CodeSPU:        "codeSPU1",
					Status:         domain.ProjectStatusUnpublished.ToUint8(),
					Labels:         []string{"标签1"},
					Desc:           "描述1",
					Utime:          1,
					Difficulties: []web.Difficulty{
						{
							Id:       1,
							Title:    "标题1",
							Status:   domain.ProjectStatusUnpublished.ToUint8(),
							Content:  "内容1",
							Analysis: "分析1",
							Utime:    1,
						},
					},
					Resumes: []web.Resume{
						{
							Id:       1,
							Role:     domain.RoleManager.ToUint8(),
							Content:  "内容1",
							Analysis: "分析1",
							Status:   domain.ResumeStatusPublished.ToUint8(),
							Utime:    1,
						},
					},
					Introductions: []web.Introduction{
						{
							Id:       1,
							Role:     domain.RoleManager.ToUint8(),
							Content:  "内容1",
							Analysis: "分析1",
							Status:   domain.IntroductionStatusPublished.ToUint8(),
							Utime:    1,
						},
					},
					Questions: []web.Question{
						{
							Id:       1,
							Analysis: "分析1",
							Answer:   "回答1",
							Title:    "标题1",
							Status:   domain.QuestionStatusPublished.ToUint8(),
							Utime:    1,
						},
					},
					Combos: []web.Combo{
						{
							Id:      1,
							Content: "内容1",
							Title:   "标题1",
							Status:  domain.ComboStatusUnpublished.ToUint8(),
							Utime:   1,
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.T().Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost,
				"/project/detail", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.Project]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
		})
	}

}

func (s *AdminProjectTestSuite) mockProject(id int64) dao.Project {
	return dao.Project{
		Id:             id,
		Title:          fmt.Sprintf("标题%d", id),
		SN:             fmt.Sprintf("SN%d", id),
		Overview:       fmt.Sprintf("概览%d", id),
		SystemDesign:   fmt.Sprintf("系统设计%d", id),
		Status:         domain.ProjectStatusUnpublished.ToUint8(),
		RefQuestionSet: id,
		GithubRepo:     fmt.Sprintf("github%d", id),
		GiteeRepo:      fmt.Sprintf("gitee%d", id),
		CodeSPU:        sqlx.NewNullString(fmt.Sprintf("codeSPU%d", id)),
		ProductSPU:     sqlx.NewNullString(fmt.Sprintf("productSPU%d", id)),
		Labels:         sqlx.JsonColumn[[]string]{Val: []string{fmt.Sprintf("标签%d", id)}, Valid: true},
		Desc:           fmt.Sprintf("描述%d", id),
		Utime:          id,
	}
}

func (s *AdminProjectTestSuite) mockQue(pid, id int64) dao.ProjectQuestion {
	return dao.ProjectQuestion{
		Id:       id,
		Pid:      pid,
		Title:    fmt.Sprintf("标题%d", id),
		Answer:   fmt.Sprintf("回答%d", id),
		Analysis: fmt.Sprintf("分析%d", id),
		Status:   domain.ResumeStatusPublished.ToUint8(),
		Utime:    id,
	}
}

func (s *AdminProjectTestSuite) mockIntr(pid, id int64) dao.ProjectIntroduction {
	return dao.ProjectIntroduction{
		Id:       id,
		Pid:      pid,
		Role:     domain.RoleManager.ToUint8(),
		Content:  fmt.Sprintf("内容%d", id),
		Analysis: fmt.Sprintf("分析%d", id),
		Status:   domain.ResumeStatusPublished.ToUint8(),
		Utime:    id,
	}
}

func (s *AdminProjectTestSuite) mockRsm(pid, id int64) dao.ProjectResume {
	return dao.ProjectResume{
		Id:       id,
		Pid:      pid,
		Role:     domain.RoleManager.ToUint8(),
		Content:  fmt.Sprintf("内容%d", id),
		Analysis: fmt.Sprintf("分析%d", id),
		Status:   domain.ResumeStatusPublished.ToUint8(),
		Utime:    id,
	}
}

func (s *AdminProjectTestSuite) mockDiff(pid, id int64) dao.ProjectDifficulty {
	return dao.ProjectDifficulty{
		Id:       id,
		Pid:      pid,
		Title:    fmt.Sprintf("标题%d", id),
		Status:   domain.ProjectStatusUnpublished.ToUint8(),
		Content:  fmt.Sprintf("内容%d", id),
		Analysis: fmt.Sprintf("分析%d", id),
		Utime:    id,
	}
}

func TestAdmin(t *testing.T) {
	suite.Run(t, new(AdminProjectTestSuite))
}
