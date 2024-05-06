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
	m := startup.InitModule()
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
				assert.Equal(t, dao.Project{
					Id:     1,
					Title:  "项目1",
					Desc:   "这是测试项目1",
					Status: domain.ProjectStatusUnpublished.ToUint8(),
					Labels: sqlx.JsonColumn[[]string]{
						Val:   []string{"标签1"},
						Valid: true,
					},
				}, prj)
			},
			req: web.Project{
				Title:  "项目1",
				Desc:   "这是测试项目1",
				Labels: []string{"标签1"},
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
					Id:     12,
					Title:  "老的标题",
					Status: domain.ProjectStatusPublished.ToUint8(),
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
					Id:     12,
					Title:  "项目1",
					Desc:   "这是测试项目1",
					Status: domain.ProjectStatusUnpublished.ToUint8(),
					Labels: sqlx.JsonColumn[[]string]{
						Val:   []string{"标签1"},
						Valid: true,
					},
					Ctime: 123,
				}, prj)
			},
			req: web.Project{
				Id:     12,
				Title:  "项目1",
				Desc:   "这是测试项目1",
				Labels: []string{"标签1"},
			},
			wantCode: 200,
			wantResp: test.Result[int64]{
				Data: 12,
			},
		},
	}

	for _, tc := range testCases {
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
				assert.Equal(t, dao.Project{
					Id:     1,
					Title:  "项目1",
					Desc:   "这是测试项目1",
					Status: domain.ProjectStatusPublished.ToUint8(),
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
				assert.Equal(t, dao.PubProject{
					Id:     1,
					Title:  "项目1",
					Desc:   "这是测试项目1",
					Status: domain.ProjectStatusPublished.ToUint8(),
					Labels: sqlx.JsonColumn[[]string]{
						Val:   []string{"标签1"},
						Valid: true,
					},
				}, pubPrj)
			},
			req: web.Project{
				Title:  "项目1",
				Desc:   "这是测试项目1",
				Labels: []string{"标签1"},
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
					Id:     12,
					Title:  "老的标题",
					Status: domain.ProjectStatusUnpublished.ToUint8(),
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
					Id:     12,
					Title:  "老的标题",
					Status: domain.ProjectStatusUnpublished.ToUint8(),
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
					Id:     12,
					Title:  "项目1",
					Desc:   "这是测试项目1",
					Status: domain.ProjectStatusPublished.ToUint8(),
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
					Id:     12,
					Title:  "项目1",
					Desc:   "这是测试项目1",
					Status: domain.ProjectStatusPublished.ToUint8(),
					Labels: sqlx.JsonColumn[[]string]{
						Val:   []string{"标签1"},
						Valid: true,
					},
					Ctime: 123,
				}, pubPrj)
			},
			req: web.Project{
				Id:     12,
				Title:  "项目1",
				Desc:   "这是测试项目1",
				Labels: []string{"标签1"},
			},
			wantCode: 200,
			wantResp: test.Result[int64]{
				Data: 12,
			},
		},
	}

	for _, tc := range testCases {
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
							Status: domain.ProjectStatusUnpublished.ToUint8(),
							Labels: []string{"标签10"},
							Desc:   "描述10",
							Utime:  10,
						},
						{
							Id:     9,
							Title:  "标题9",
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
					Id:     1,
					Title:  "标题1",
					Status: domain.ProjectStatusUnpublished.ToUint8(),
					Labels: []string{"标签1"},
					Desc:   "描述1",
					Utime:  1,
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
							Status:   domain.ResumeStatusPublished.ToUint8(),
							Utime:    1,
						},
					},
					Questions: []web.Question{
						{
							Id:       1,
							Analysis: "分析1",
							Answer:   "回答1",
							Title:    "标题1",
							Status:   domain.ResumeStatusPublished.ToUint8(),
							Utime:    1,
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
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
		Id:     id,
		Title:  fmt.Sprintf("标题%d", id),
		Status: domain.ProjectStatusUnpublished.ToUint8(),
		Labels: sqlx.JsonColumn[[]string]{Val: []string{fmt.Sprintf("标签%d", id)}, Valid: true},
		Desc:   fmt.Sprintf("描述%d", id),
		Utime:  id,
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
