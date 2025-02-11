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

	"github.com/ecodeclub/webook/internal/permission"
	permissionmocks "github.com/ecodeclub/webook/internal/permission/mocks"

	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/interactive"
	intrmocks "github.com/ecodeclub/webook/internal/interactive/mocks"
	"github.com/gin-gonic/gin"
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

type ProjectTestSuite struct {
	suite.Suite
	hdl     *project.Handler
	server  *egin.Component
	db      *egorm.Component
	prjDAO  dao.ProjectDAO
	permSvc *permissionmocks.MockService
}

func (s *ProjectTestSuite) SetupSuite() {
	ctrl := gomock.NewController(s.T())
	intrSvc := intrmocks.NewMockService(ctrl)
	intrModule := &interactive.Module{
		Svc: intrSvc,
	}

	// 模拟返回的数据
	// 使用如下规律:
	// 1. liked == id % 2 == 1 (奇数为 true)
	// 2. collected = id %2 == 0 (偶数为 true)
	// 3. viewCnt = id + 1
	// 4. likeCnt = id + 2
	// 5. collectCnt = id + 3
	intrSvc.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		AnyTimes().DoAndReturn(func(ctx context.Context,
		biz string, id int64, uid int64) (interactive.Interactive, error) {
		intr := s.mockInteractive(biz, id)
		return intr, nil
	})
	intrSvc.EXPECT().GetByIds(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context,
		biz string, uid int64, ids []int64) (map[int64]interactive.Interactive, error) {
		res := make(map[int64]interactive.Interactive, len(ids))
		for _, id := range ids {
			intr := s.mockInteractive(biz, id)
			res[id] = intr
		}
		return res, nil
	}).AnyTimes()

	permSvc := permissionmocks.NewMockService(ctrl)
	permModule := &permission.Module{
		Svc: permSvc,
	}
	s.permSvc = permSvc
	m, err := startup.InitModule(intrModule, permModule, session.DefaultProvider())
	require.NoError(s.T(), err)
	s.hdl = m.Hdl

	econf.Set("server", map[string]any{"contextTimeout": "10s"})
	server := egin.Load("server").Build()
	s.hdl.PublicRoutes(server.Engine)
	server.Use(func(ctx *gin.Context) {
		ctx.Set(session.CtxSessionKey, session.NewMemorySession(session.Claims{
			Uid: 123,
		}))
	})
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
					Total: 5,
					Projects: []web.Project{
						{
							Id:         9,
							SN:         "SN9",
							Title:      "标题9",
							Status:     domain.ProjectStatusPublished.ToUint8(),
							Labels:     []string{"标签9"},
							Desc:       "描述9",
							Utime:      9,
							CodeSPU:    "code-spu-9",
							ProductSPU: "product-spu-9",
							Interactive: web.Interactive{
								ViewCnt:    10,
								LikeCnt:    11,
								CollectCnt: 12,
								Liked:      true,
								Collected:  false,
							},
						},
						{
							Id:         7,
							SN:         "SN7",
							Title:      "标题7",
							Status:     domain.ProjectStatusPublished.ToUint8(),
							Labels:     []string{"标签7"},
							Desc:       "描述7",
							Utime:      7,
							CodeSPU:    "code-spu-7",
							ProductSPU: "product-spu-7",
							Interactive: web.Interactive{
								ViewCnt:    8,
								LikeCnt:    9,
								CollectCnt: 10,
								Liked:      true,
								Collected:  false,
							},
						},
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
			wantResp: test.Result[web.ProjectList]{
				Data: web.ProjectList{
					Total: 5,
					Projects: []web.Project{
						{
							Id:         1,
							SN:         "SN1",
							Title:      "标题1",
							Status:     domain.ProjectStatusPublished.ToUint8(),
							Labels:     []string{"标签1"},
							Desc:       "描述1",
							Utime:      1,
							CodeSPU:    "code-spu-1",
							ProductSPU: "product-spu-1",
							Interactive: web.Interactive{
								ViewCnt:    2,
								LikeCnt:    3,
								CollectCnt: 4,
								Liked:      true,
								Collected:  false,
							},
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

func (s *ProjectTestSuite) TestProjectDetail() {
	s.insertWholeProject(1)
	s.insertWholeProject(3)

	s.permSvc.EXPECT().HasPermission(gomock.Any(), permission.Permission{
		Biz:   "project",
		BizID: 1,
		Uid:   123,
	}).Return(true, nil)

	s.permSvc.EXPECT().HasPermission(gomock.Any(), permission.Permission{
		Biz:   "project",
		BizID: 3,
		Uid:   123,
	}).Return(false, nil)

	testCases := []struct {
		name string
		req  web.IdReq

		wantCode int
		wantResp test.Result[web.Project]
	}{
		{
			name:     "有权限",
			req:      web.IdReq{Id: 1},
			wantCode: 200,
			wantResp: test.Result[web.Project]{
				Data: web.Project{
					Id:             1,
					SN:             "SN1",
					Title:          "标题1",
					Overview:       "概览1",
					SystemDesign:   "系统设计1",
					RefQuestionSet: 1,
					GithubRepo:     "github1",
					GiteeRepo:      "gitee1",
					Status:         domain.ProjectStatusPublished.ToUint8(),
					Labels:         []string{"标签1"},
					Desc:           "描述1",
					Utime:          1,
					Permitted:      true,
					CodeSPU:        "code-spu-1",
					ProductSPU:     "product-spu-1",
					Interactive: web.Interactive{
						ViewCnt:    2,
						LikeCnt:    3,
						CollectCnt: 4,
						Liked:      true,
						Collected:  false,
					},
					Difficulties: []web.Difficulty{
						{
							Id:       11,
							Title:    "标题11",
							Status:   domain.ProjectStatusPublished.ToUint8(),
							Content:  "内容11",
							Analysis: "分析11",
							Utime:    11,
						},
					},
					Resumes: []web.Resume{
						{
							Id:       11,
							Role:     domain.RoleManager.ToUint8(),
							Content:  "内容11",
							Analysis: "分析11",
							Status:   domain.ResumeStatusPublished.ToUint8(),
							Utime:    11,
						},
					},
					Introductions: []web.Introduction{
						{
							Id:       11,
							Role:     domain.RoleManager.ToUint8(),
							Content:  "内容11",
							Analysis: "分析11",
							Status:   domain.ResumeStatusPublished.ToUint8(),
							Utime:    11,
						},
					},
					Questions: []web.Question{
						{
							Id:       11,
							Analysis: "分析11",
							Answer:   "回答11",
							Title:    "标题11",
							Status:   domain.ResumeStatusPublished.ToUint8(),
							Utime:    11,
						},
					},
					Combos: []web.Combo{
						{
							Id:      11,
							Content: "内容11",
							Title:   "标题11",
							Status:  domain.ResumeStatusPublished.ToUint8(),
							Utime:   11,
						},
					},
				},
			},
		},
		{
			name:     "无权限",
			req:      web.IdReq{Id: 3},
			wantCode: 200,
			wantResp: test.Result[web.Project]{
				Data: web.Project{
					Id:         3,
					SN:         "SN3",
					Title:      "标题3",
					Status:     domain.ProjectStatusPublished.ToUint8(),
					Labels:     []string{"标签3"},
					Desc:       "描述3",
					Utime:      3,
					CodeSPU:    "code-spu-3",
					ProductSPU: "product-spu-3",
					Interactive: web.Interactive{
						ViewCnt:    4,
						LikeCnt:    5,
						CollectCnt: 6,
						Liked:      true,
						Collected:  false,
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

func (s *ProjectTestSuite) insertWholeProject(id int64) {
	// 插入各种数据
	prj := s.mockProject(id)
	err := s.db.Create(&prj).Error
	require.NoError(s.T(), err)
	// 难点
	diff := s.mockDiff(id, id*10+1)
	err = s.db.Create(&diff).Error
	require.NoError(s.T(), err)
	// 简历
	rsm := s.mockRsm(id, id*10+1)
	err = s.db.Create(&rsm).Error
	require.NoError(s.T(), err)
	// 项目介绍
	intr := s.mockIntr(id, id*10+1)
	err = s.db.Create(&intr).Error
	require.NoError(s.T(), err)

	que := s.mockQue(id, id*10+1)
	err = s.db.Create(&que).Error
	require.NoError(s.T(), err)

	combo := s.mockCombo(id, id*10+1)
	err = s.db.Create(&combo).Error
	require.NoError(s.T(), err)
}

func (s *ProjectTestSuite) mockCombo(pid, id int64) dao.PubProjectCombo {
	return dao.PubProjectCombo{
		Id:      id,
		Pid:     pid,
		Title:   fmt.Sprintf("标题%d", id),
		Content: fmt.Sprintf("内容%d", id),
		Status:  domain.ComboStatusPublished.ToUint8(),
		Ctime:   id,
		Utime:   id,
	}
}

func (s *ProjectTestSuite) mockProject(id int64) dao.PubProject {
	return dao.PubProject{
		Id:             id,
		SN:             fmt.Sprintf("SN%d", id),
		Title:          fmt.Sprintf("标题%d", id),
		Overview:       fmt.Sprintf("概览%d", id),
		SystemDesign:   fmt.Sprintf("系统设计%d", id),
		RefQuestionSet: id,
		GithubRepo:     fmt.Sprintf("github%d", id),
		GiteeRepo:      fmt.Sprintf("gitee%d", id),
		Status:         uint8(id%2 + 1),
		Labels:         sqlx.JsonColumn[[]string]{Val: []string{fmt.Sprintf("标签%d", id)}, Valid: true},
		Desc:           fmt.Sprintf("描述%d", id),
		Utime:          id,
		ProductSPU:     sqlx.NewNullString(fmt.Sprintf("product-spu-%d", id)),
		CodeSPU:        sqlx.NewNullString(fmt.Sprintf("code-spu-%d", id)),
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

func (s *ProjectTestSuite) mockInteractive(biz string, id int64) interactive.Interactive {
	liked := id%2 == 1
	collected := id%2 == 0
	return interactive.Interactive{
		Biz:        biz,
		BizId:      id,
		ViewCnt:    int(id + 1),
		LikeCnt:    int(id + 2),
		CollectCnt: int(id + 3),
		Liked:      liked,
		Collected:  collected,
	}
}

func TestProject(t *testing.T) {
	suite.Run(t, new(ProjectTestSuite))
}
