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
	"fmt"
	"net/http"
	"testing"

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

type ProjectTestSuite struct {
	suite.Suite
	hdl    *project.Handler
	server *egin.Component
	db     *egorm.Component
	prjDAO dao.ProjectDAO
}

func (s *ProjectTestSuite) SetupSuite() {
	m := startup.InitModule()
	s.hdl = m.Hdl

	econf.Set("server", map[string]any{"contextTimeout": "10s"})
	server := egin.Load("server").Build()
	s.hdl.PrivateRoutes(server.Engine)
	s.server = server
	s.db = testioc.InitDB()
	s.prjDAO = dao.NewGORMProjectDAO(s.db)
}

func (s *ProjectTestSuite) TearDownTest() {
	err := s.db.Exec("TRUNCATE TABLE pub_projects;").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE pub_project_difficulties;").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE pub_project_resumes;").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE pub_project_questions;").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE pub_project_introductions;").Error
	require.NoError(s.T(), err)
}

func (s *ProjectTestSuite) TestProjectList() {
	prjs := make([]dao.PubProject, 0, 10)
	for i := 0; i < 10; i++ {
		prjs = append(prjs, s.mockProject(int64(i+1)))
	}
	err := s.db.Create(&prjs).Error
	require.NoError(s.T(), err)
	testCases := []struct {
		name string
		req  web.Page

		wantCode int
		wantResp test.Result[[]web.Project]
	}{
		{
			name: "从头获取成功",
			req: web.Page{
				Offset: 0,
				Limit:  2,
			},
			wantCode: 200,
			wantResp: test.Result[[]web.Project]{
				Data: []web.Project{
					{
						Id:     9,
						Title:  "标题9",
						Status: domain.ProjectStatusPublished.ToUint8(),
						Labels: []string{"标签9"},
						Desc:   "描述9",
						Utime:  9,
					},
					{
						Id:     7,
						Title:  "标题7",
						Status: domain.ProjectStatusPublished.ToUint8(),
						Labels: []string{"标签7"},
						Desc:   "描述7",
						Utime:  7,
					},
				},
			},
		},
		{
			name: "末尾部分获取",
			req: web.Page{
				Offset: 4,
				Limit:  2,
			},
			wantCode: 200,
			wantResp: test.Result[[]web.Project]{
				Data: []web.Project{
					{
						Id:     1,
						Title:  "标题1",
						Status: domain.ProjectStatusPublished.ToUint8(),
						Labels: []string{"标签1"},
						Desc:   "描述1",
						Utime:  1,
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
			recorder := test.NewJSONResponseRecorder[[]web.Project]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
		})
	}
}

func (s *ProjectTestSuite) TestProjectDetail() {
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
					Status: domain.ProjectStatusPublished.ToUint8(),
					Labels: []string{"标签1"},
					Desc:   "描述1",
					Utime:  1,
					Difficulties: []web.Difficulty{
						{
							Id:       1,
							Title:    "标题1",
							Status:   domain.ProjectStatusPublished.ToUint8(),
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

func (s *ProjectTestSuite) mockProject(id int64) dao.PubProject {
	return dao.PubProject{
		Id:     id,
		Title:  fmt.Sprintf("标题%d", id),
		Status: uint8(id%2 + 1),
		Labels: sqlx.JsonColumn[[]string]{Val: []string{fmt.Sprintf("标签%d", id)}, Valid: true},
		Desc:   fmt.Sprintf("描述%d", id),
		Utime:  id,
	}
}

func (s *ProjectTestSuite) mockQue(pid, id int64) dao.PubProjectQuestion {
	return dao.PubProjectQuestion{
		Id:       id,
		Pid:      pid,
		Title:    fmt.Sprintf("标题%d", id),
		Answer:   fmt.Sprintf("回答%d", id),
		Analysis: fmt.Sprintf("分析%d", id),
		Status:   domain.ResumeStatusPublished.ToUint8(),
		Utime:    id,
	}
}

func (s *ProjectTestSuite) mockIntr(pid, id int64) dao.PubProjectIntroduction {
	return dao.PubProjectIntroduction{
		Id:       id,
		Pid:      pid,
		Role:     domain.RoleManager.ToUint8(),
		Content:  fmt.Sprintf("内容%d", id),
		Analysis: fmt.Sprintf("分析%d", id),
		Status:   domain.ResumeStatusPublished.ToUint8(),
		Utime:    id,
	}
}

func (s *ProjectTestSuite) mockRsm(pid, id int64) dao.PubProjectResume {
	return dao.PubProjectResume{
		Id:       id,
		Pid:      pid,
		Role:     domain.RoleManager.ToUint8(),
		Content:  fmt.Sprintf("内容%d", id),
		Analysis: fmt.Sprintf("分析%d", id),
		Status:   domain.ResumeStatusPublished.ToUint8(),
		Utime:    id,
	}
}

func (s *ProjectTestSuite) mockDiff(pid, id int64) dao.PubProjectDifficulty {
	return dao.PubProjectDifficulty{
		Id:       id,
		Pid:      pid,
		Title:    fmt.Sprintf("标题%d", id),
		Status:   domain.ProjectStatusPublished.ToUint8(),
		Content:  fmt.Sprintf("内容%d", id),
		Analysis: fmt.Sprintf("分析%d", id),
		Utime:    id,
	}
}

func TestProject(t *testing.T) {
	suite.Run(t, new(ProjectTestSuite))
}
