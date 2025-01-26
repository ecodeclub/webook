//go:build e2e

package integration

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"github.com/ecodeclub/webook/internal/ai"

	"github.com/ecodeclub/ekit/iox"
	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/cases"
	casemocks "github.com/ecodeclub/webook/internal/cases/mocks"
	"github.com/ecodeclub/webook/internal/resume/internal/integration/startup"
	"github.com/ecodeclub/webook/internal/resume/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/resume/internal/web"
	"github.com/ecodeclub/webook/internal/test"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/ego-component/egorm"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/econf"
	"github.com/gotomicro/ego/server/egin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"gorm.io/gorm"
)

type ProjectTestSuite struct {
	suite.Suite
	db     *egorm.Component
	server *egin.Component
	hdl    *web.ProjectHandler
	pdao   dao.ResumeProjectDAO
}

func (s *ProjectTestSuite) TearDownTest() {
	err := s.db.Exec("TRUNCATE  TABLE `resume_projects`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE `contributions`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE  TABLE `difficulties`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE  TABLE `ref_cases`").Error
	require.NoError(s.T(), err)
}

func (s *ProjectTestSuite) SetupSuite() {
	ctrl := gomock.NewController(s.T())
	examSvc := casemocks.NewMockExamineService(ctrl)
	examSvc.EXPECT().GetResults(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, uid int64, ids []int64) (map[int64]cases.ExamineResult, error) {
		res := slice.Map(ids, func(idx int, src int64) cases.ExamineResult {
			return cases.ExamineResult{
				Cid:    src,
				Result: cases.ExamineResultEnum(src % 4),
			}
		})
		resMap := make(map[int64]cases.ExamineResult, len(res))
		for _, examRes := range res {
			resMap[examRes.Cid] = examRes
		}
		return resMap, nil
	}).AnyTimes()
	caseSvc := casemocks.NewMockService(ctrl)
	caseSvc.EXPECT().GetPubByIDs(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, ids []int64) ([]cases.Case, error) {
			return slice.Map(ids, func(idx int, src int64) cases.Case {
				return cases.Case{
					Id:           src,
					Title:        "这是案例" + strconv.FormatInt(src, 10),
					Introduction: "这是案例的简介" + strconv.FormatInt(src, 10),
				}
			}), nil
		}).AnyTimes()

	module := startup.InitModule(&cases.Module{
		ExamineSvc: examSvc,
		Svc:        caseSvc,
	},
		&ai.Module{})
	econf.Set("server", map[string]any{"contextTimeout": "1s"})
	server := egin.Load("server").Build()
	server.Use(func(ctx *gin.Context) {
		ctx.Set("_session", session.NewMemorySession(session.Claims{
			Uid:  uid,
			Data: map[string]string{"creator": "true"},
		}))
	})
	module.PrjHdl.MemberRoutes(server.Engine)
	s.server = server
	s.db = testioc.InitDB()
	err := dao.InitTables(s.db)
	require.NoError(s.T(), err)
	s.pdao = dao.NewResumeProjectDAO(s.db)
}

func TestResumeModule(t *testing.T) {
	suite.Run(t, new(ProjectTestSuite))
}

func (s *ProjectTestSuite) TestSaveResumeProject() {
	testcases := []struct {
		name     string
		req      web.SaveProjectReq
		before   func(t *testing.T)
		after    func(t *testing.T)
		wantCode int
		wantResp test.Result[int64]
	}{
		{
			name: "新建",
			req: web.SaveProjectReq{
				Project: web.Project{
					StartTime:    1,
					EndTime:      321,
					Name:         "project",
					Introduction: "introduction",
					Core:         true,
				},
			},
			before: func(t *testing.T) {
			},
			after: func(t *testing.T) {
				project, err := s.pdao.First(context.Background(), 1)
				require.NoError(t, err)
				require.True(t, project.Ctime != 0)
				require.True(t, project.Utime != 0)
				project.Ctime = 0
				project.Utime = 0
				assert.Equal(t, dao.ResumeProject{
					ID:           1,
					StartTime:    1,
					EndTime:      321,
					Uid:          uid,
					Name:         "project",
					Introduction: "introduction",
					Core:         true,
				}, project)
			},
			wantResp: test.Result[int64]{
				Data: 1,
			},
			wantCode: 200,
		},
		{
			name: "更新",
			req: web.SaveProjectReq{
				Project: web.Project{
					Id:           1,
					StartTime:    2,
					EndTime:      666,
					Uid:          uid,
					Name:         "projectnew",
					Introduction: "introductionnew",
					Core:         false,
				},
			},
			before: func(t *testing.T) {
				_, err := s.pdao.Upsert(context.Background(), dao.ResumeProject{
					StartTime:    1,
					Uid:          uid,
					EndTime:      321,
					Name:         "project",
					Introduction: "introduction",
					Core:         true,
					Utime:        1,
					Ctime:        2,
				})
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				project, err := s.pdao.First(context.Background(), 1)
				require.NoError(t, err)
				require.True(t, project.Ctime != 0)
				require.True(t, project.Utime != 0)
				project.Ctime = 0
				project.Utime = 0
				assert.Equal(t, dao.ResumeProject{
					ID:           1,
					StartTime:    2,
					EndTime:      666,
					Uid:          uid,
					Name:         "projectnew",
					Introduction: "introductionnew",
					Core:         false,
				}, project)
			},
			wantResp: test.Result[int64]{
				Data: 1,
			},
			wantCode: 200,
		},
	}
	for _, tc := range testcases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/resume/project/save", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[int64]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
			tc.after(t)
			// 清理数据
			err = s.db.Exec("TRUNCATE  TABLE `resume_projects`").Error
			require.NoError(t, err)
			err = s.db.Exec("TRUNCATE TABLE `contributions`").Error
			require.NoError(s.T(), err)
		})
	}
}

func (s *ProjectTestSuite) TestSaveContribution() {
	testcases := []struct {
		name     string
		req      web.SaveContributionReq
		before   func(t *testing.T)
		after    func(t *testing.T)
		wantCode int
	}{
		{
			name: "新建有case的贡献",
			req: web.SaveContributionReq{
				ID: 1,
				Contribution: web.Contribution{
					Type: "stability",
					Desc: "stability_desc",
					RefCases: []web.Case{
						{
							Id:        1,
							Highlight: true,
							Level:     0,
						},
						{
							Id:        2,
							Highlight: false,
							Level:     1,
						},
					},
				},
			},
			before: func(t *testing.T) {
				_, err := s.pdao.Upsert(context.Background(), dao.ResumeProject{
					ID:           1,
					StartTime:    2,
					EndTime:      666,
					Uid:          uid,
					Name:         "projectnew",
					Introduction: "introductionnew",
					Core:         false,
				})
				require.NoError(t, err)
			},
			wantCode: 200,
			after: func(t *testing.T) {
				var contribution dao.Contribution
				err := s.db.WithContext(context.Background()).Where("id = ?", 1).
					First(&contribution).Error
				require.NoError(t, err)
				s.assertContribution(&contribution, &dao.Contribution{
					ID:        1,
					Type:      "stability",
					Desc:      "stability_desc",
					ProjectID: 1,
				})
				var refCases []dao.RefCase
				err = s.db.WithContext(context.Background()).
					Where("contribution_id = ?", 1).
					Order("id desc").
					Find(&refCases).Error
				require.NoError(t, err)
				for idx := range refCases {
					require.True(t, refCases[idx].Ctime != 0)
					require.True(t, refCases[idx].Utime != 0)
					refCases[idx].Ctime = 0
					refCases[idx].Utime = 0
				}
				assert.Equal(t, []dao.RefCase{
					{
						ID:             2,
						ContributionID: 1,
						CaseID:         2,
						Highlight:      false,
						Level:          1,
					},
					{
						ID:             1,
						ContributionID: 1,
						CaseID:         1,
						Highlight:      true,
						Level:          0,
					},
				}, refCases)

			},
		},
		{
			name: "更新case",
			req: web.SaveContributionReq{
				ID: 1,
				Contribution: web.Contribution{
					ID:   1,
					Type: "stability",
					Desc: "stability_desc",
					RefCases: []web.Case{
						{
							Id:        2,
							Highlight: true,
							Level:     0,
						},
						{
							Id:        3,
							Highlight: false,
							Level:     1,
						},
					},
				},
			},
			before: func(t *testing.T) {
				_, err := s.pdao.Upsert(context.Background(), dao.ResumeProject{
					ID:           1,
					StartTime:    2,
					EndTime:      666,
					Uid:          uid,
					Name:         "projectnew",
					Introduction: "introductionnew",
					Core:         false,
				})
				require.NoError(t, err)
				_, err = s.pdao.SaveContribution(context.Background(), dao.Contribution{
					ID:        1,
					Type:      "type",
					ProjectID: 1,
					Desc:      "desc",
				}, []dao.RefCase{
					{
						ID:             1,
						CaseID:         1,
						Highlight:      true,
						ContributionID: 1,
					},
					{
						ID:             2,
						CaseID:         2,
						Highlight:      false,
						ContributionID: 1,
					},
				})
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				var contribution dao.Contribution
				err := s.db.WithContext(context.Background()).Where("id = ?", 1).
					First(&contribution).Error
				require.NoError(t, err)
				s.assertContribution(&contribution, &dao.Contribution{
					ID:        1,
					Type:      "type",
					Desc:      "stability_desc",
					ProjectID: 1,
				})
				var refCases []dao.RefCase
				err = s.db.WithContext(context.Background()).
					Where("contribution_id = ?", 1).
					Order("id desc").
					Find(&refCases).Error
				require.NoError(t, err)
				for idx := range refCases {
					require.True(t, refCases[idx].Ctime != 0)
					require.True(t, refCases[idx].Utime != 0)
					require.True(t, refCases[idx].ID != 0)
					refCases[idx].Ctime = 0
					refCases[idx].Utime = 0
					refCases[idx].ID = 0
				}
				assert.Equal(t, []dao.RefCase{
					{
						ContributionID: 1,
						CaseID:         3,
						Highlight:      false,
						Level:          1,
					},
					{
						ContributionID: 1,
						CaseID:         2,
						Highlight:      true,
						Level:          0,
					},
				}, refCases)
			},
			wantCode: 200,
		},
		{
			wantCode: 200,
			name:     "添加没有case的贡献",
			req: web.SaveContributionReq{
				ID: 1,
				Contribution: web.Contribution{
					Type: "stability",
					Desc: "stability_desc",
				},
			},
			before: func(t *testing.T) {
				_, err := s.pdao.Upsert(context.Background(), dao.ResumeProject{
					ID:           1,
					StartTime:    2,
					EndTime:      666,
					Uid:          uid,
					Name:         "projectnew",
					Introduction: "introductionnew",
					Core:         false,
				})
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				var contribution dao.Contribution
				err := s.db.WithContext(context.Background()).Where("id = ?", 1).
					First(&contribution).Error
				require.NoError(t, err)
				s.assertContribution(&contribution, &dao.Contribution{
					ID:        1,
					Type:      "stability",
					Desc:      "stability_desc",
					ProjectID: 1,
				})
				var refCases []dao.RefCase
				err = s.db.WithContext(context.Background()).
					Where("contribution_id = ?", 1).
					Order("id desc").
					Find(&refCases).Error
				require.NoError(t, err)
				assert.Equal(t, 0, len(refCases))

			},
		},
		{
			wantCode: 200,
			name:     "添加没有case的贡献,原来有",
			req: web.SaveContributionReq{
				ID: 1,
				Contribution: web.Contribution{
					ID:   1,
					Type: "type",
					Desc: "stability_desc",
				},
			},
			before: func(t *testing.T) {
				_, err := s.pdao.Upsert(context.Background(), dao.ResumeProject{
					ID:           1,
					StartTime:    2,
					EndTime:      666,
					Uid:          uid,
					Name:         "projectnew",
					Introduction: "introductionnew",
					Core:         false,
				})
				require.NoError(t, err)
				_, err = s.pdao.SaveContribution(context.Background(), dao.Contribution{
					ID:        1,
					Type:      "type",
					ProjectID: 1,
					Desc:      "desc",
				}, []dao.RefCase{})
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				var contribution dao.Contribution
				err := s.db.WithContext(context.Background()).Where("id = ?", 1).
					First(&contribution).Error
				require.NoError(t, err)
				s.assertContribution(&contribution, &dao.Contribution{
					ID:        1,
					Type:      "type",
					Desc:      "stability_desc",
					ProjectID: 1,
				})
				var refCases []dao.RefCase
				err = s.db.WithContext(context.Background()).
					Where("contribution_id = ?", 1).
					Order("id desc").
					Find(&refCases).Error
				require.NoError(t, err)
				assert.Equal(t, 0, len(refCases))
			},
		},
	}
	for _, tc := range testcases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/resume/project/contribution/save", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[int64]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			tc.after(t)
			// 清理数据
			err = s.db.Exec("TRUNCATE  TABLE `resume_projects`").Error
			require.NoError(t, err)
			err = s.db.Exec("TRUNCATE TABLE `contributions`").Error
			require.NoError(s.T(), err)
			err = s.db.Exec("TRUNCATE  TABLE `ref_cases`").Error
			require.NoError(s.T(), err)
		})
	}
}

func (s *ProjectTestSuite) TestSaveDifficulty() {
	testcases := []struct {
		name     string
		req      web.SaveDifficultyReq
		before   func(t *testing.T)
		after    func(t *testing.T)
		wantCode int
	}{
		{
			wantCode: 200,
			name:     "新增",
			req: web.SaveDifficultyReq{
				ID: 1,
				Difficulty: web.Difficulty{
					Desc: "desc",
					Case: web.Case{
						Id:    1,
						Level: 1,
					},
				},
			},
			before: func(t *testing.T) {
			},
			after: func(t *testing.T) {
				var actual dao.Difficulty
				err := s.db.WithContext(context.Background()).Where("id = ?", 1).
					First(&actual).Error
				require.NoError(t, err)
				require.True(t, actual.Ctime != 0)
				require.True(t, actual.Utime != 0)
				actual.Ctime = 0
				actual.Utime = 0
				assert.Equal(t, dao.Difficulty{
					ID:        1,
					Desc:      "desc",
					CaseID:    1,
					ProjectID: 1,
					Level:     1,
				}, actual)
			},
		},
		{
			name:     "更新",
			wantCode: 200,
			req: web.SaveDifficultyReq{
				ID: 1,
				Difficulty: web.Difficulty{
					ID:   1,
					Desc: "desc_new",
					Case: web.Case{
						Id:    2,
						Level: 3,
					},
				},
			},
			before: func(t *testing.T) {
				_, err := s.pdao.Upsert(context.Background(), dao.ResumeProject{
					ID:           1,
					StartTime:    2,
					EndTime:      666,
					Uid:          uid,
					Name:         "projectnew",
					Introduction: "introductionnew",
					Core:         false,
				})
				require.NoError(t, err)
				err = s.pdao.SaveDifficulty(context.Background(), dao.Difficulty{
					ID:        1,
					ProjectID: 1,
					CaseID:    2,
					Level:     1,
					Desc:      "desc",
				})
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				var actual dao.Difficulty
				err := s.db.WithContext(context.Background()).Where("id = ?", 1).
					First(&actual).Error
				require.NoError(t, err)
				require.True(t, actual.Ctime != 0)
				require.True(t, actual.Utime != 0)
				actual.Ctime = 0
				actual.Utime = 0
				assert.Equal(t, dao.Difficulty{
					ID:        1,
					Desc:      "desc_new",
					CaseID:    2,
					ProjectID: 1,
					Level:     3,
				}, actual)
			},
		},
	}
	for _, tc := range testcases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/resume/project/difficulty/save", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[int64]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			tc.after(t)
			// 清理数据
			err = s.db.Exec("TRUNCATE  TABLE `resume_projects`").Error
			require.NoError(t, err)
			err = s.db.Exec("TRUNCATE TABLE `difficulties`").Error
			require.NoError(s.T(), err)
		})
	}
}

func (s *ProjectTestSuite) TestDeleteResumeProject() {
	testcases := []struct {
		name     string
		before   func(t *testing.T)
		req      web.IDItem
		after    func(t *testing.T)
		wantCode int
	}{
		{
			name: "成功删除",
			before: func(t *testing.T) {
				_, err := s.pdao.Upsert(context.Background(), dao.ResumeProject{
					ID:           1,
					StartTime:    2,
					EndTime:      666,
					Uid:          uid,
					Name:         "projectnew",
					Introduction: "introductionnew",
					Core:         false,
				})
				require.NoError(t, err)
				err = s.pdao.SaveDifficulty(context.Background(), dao.Difficulty{
					ID:        1,
					ProjectID: 1,
					CaseID:    2,
					Level:     1,
					Desc:      "desc",
				})
				require.NoError(t, err)
				_, err = s.pdao.SaveContribution(context.Background(), dao.Contribution{
					ID:        1,
					Type:      "type",
					ProjectID: 1,
					Desc:      "desc",
				}, []dao.RefCase{
					{
						ID:             1,
						CaseID:         2,
						Highlight:      true,
						ContributionID: 1,
					},
					{
						ID:             2,
						CaseID:         3,
						Highlight:      false,
						ContributionID: 1,
					},
				})
				require.NoError(t, err)
			},
			req: web.IDItem{
				ID: 1,
			},
			after: func(t *testing.T) {
				_, err := s.pdao.First(context.Background(), 1)
				assert.Error(t, gorm.ErrRecordNotFound, err)
				contributions, err := s.pdao.FindContributions(context.Background(), 1)
				require.NoError(t, err)
				assert.Equal(t, 0, len(contributions))
				diffculties, err := s.pdao.FindDifficulties(context.Background(), 1)
				require.NoError(t, err)
				assert.Equal(t, 0, len(diffculties))
			},
			wantCode: 200,
		},
	}
	for _, tc := range testcases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/resume/project/delete", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[int64]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			tc.after(t)
			// 清理数据
			err = s.db.Exec("TRUNCATE  TABLE `resume_projects`").Error
			require.NoError(s.T(), err)
			err = s.db.Exec("TRUNCATE TABLE `contributions`").Error
			require.NoError(s.T(), err)
			err = s.db.Exec("TRUNCATE  TABLE `difficulties`").Error
			require.NoError(s.T(), err)
			err = s.db.Exec("TRUNCATE  TABLE `ref_cases`").Error
			require.NoError(s.T(), err)
		})
	}

}

func (s *ProjectTestSuite) TestDeleteDifficulty() {
	testcases := []struct {
		name     string
		before   func(t *testing.T)
		req      web.IDItem
		after    func(t *testing.T)
		wantCode int
	}{
		{
			name: "成功删除",
			before: func(t *testing.T) {
				_, err := s.pdao.Upsert(context.Background(), dao.ResumeProject{
					ID:           1,
					StartTime:    2,
					EndTime:      666,
					Uid:          uid,
					Name:         "projectnew",
					Introduction: "introductionnew",
					Core:         false,
				})
				require.NoError(t, err)
				err = s.pdao.SaveDifficulty(context.Background(), dao.Difficulty{
					ID:        1,
					ProjectID: 1,
					CaseID:    2,
					Level:     1,
					Desc:      "desc",
				})
				require.NoError(t, err)
				_, err = s.pdao.SaveContribution(context.Background(), dao.Contribution{
					ID:        1,
					Type:      "type",
					ProjectID: 1,
					Desc:      "desc",
				}, []dao.RefCase{
					{
						ID:             1,
						CaseID:         2,
						Highlight:      true,
						ContributionID: 1,
					},
					{
						ID:             2,
						CaseID:         3,
						Highlight:      false,
						ContributionID: 1,
					},
				})
				require.NoError(t, err)
			},
			req: web.IDItem{
				ID: 1,
			},
			after: func(t *testing.T) {
				_, err := s.pdao.First(context.Background(), 1)
				require.NoError(t, err)
				contributions, err := s.pdao.FindContributions(context.Background(), 1)
				require.NoError(t, err)
				assert.Equal(t, 1, len(contributions))
				diffculties, err := s.pdao.FindDifficulties(context.Background(), 1)
				require.NoError(t, err)
				assert.Equal(t, 0, len(diffculties))
			},
			wantCode: 200,
		},
	}
	for _, tc := range testcases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/resume/project/difficulty/del", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[int64]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			tc.after(t)
			// 清理数据
			err = s.db.Exec("TRUNCATE  TABLE `resume_projects`").Error
			require.NoError(s.T(), err)
			err = s.db.Exec("TRUNCATE TABLE `contributions`").Error
			require.NoError(s.T(), err)
			err = s.db.Exec("TRUNCATE  TABLE `difficulties`").Error
			require.NoError(s.T(), err)
			err = s.db.Exec("TRUNCATE  TABLE `ref_cases`").Error
			require.NoError(s.T(), err)
		})
	}
}

func (s *ProjectTestSuite) TestDeleteContribution() {
	testcases := []struct {
		name     string
		before   func(t *testing.T)
		req      web.IDItem
		after    func(t *testing.T)
		wantCode int
	}{
		{
			name: "成功删除",
			before: func(t *testing.T) {
				_, err := s.pdao.Upsert(context.Background(), dao.ResumeProject{
					ID:           1,
					StartTime:    2,
					EndTime:      666,
					Uid:          uid,
					Name:         "projectnew",
					Introduction: "introductionnew",
					Core:         false,
				})
				require.NoError(t, err)
				err = s.pdao.SaveDifficulty(context.Background(), dao.Difficulty{
					ID:        1,
					ProjectID: 1,
					CaseID:    2,
					Level:     1,
					Desc:      "desc",
				})
				require.NoError(t, err)
				_, err = s.pdao.SaveContribution(context.Background(), dao.Contribution{
					ID:        1,
					Type:      "type",
					ProjectID: 1,
					Desc:      "desc",
				}, []dao.RefCase{
					{
						ID:             1,
						CaseID:         2,
						Highlight:      true,
						ContributionID: 1,
					},
					{
						ID:             2,
						CaseID:         3,
						Highlight:      false,
						ContributionID: 1,
					},
				})
				require.NoError(t, err)
			},
			req: web.IDItem{
				ID: 1,
			},
			after: func(t *testing.T) {
				_, err := s.pdao.First(context.Background(), 1)
				require.NoError(t, err)
				contributions, err := s.pdao.FindContributions(context.Background(), 1)
				require.NoError(t, err)
				assert.Equal(t, 0, len(contributions))
				diffculties, err := s.pdao.FindDifficulties(context.Background(), 1)
				require.NoError(t, err)
				assert.Equal(t, 1, len(diffculties))
				var ids []int64
				err = s.db.WithContext(context.Background()).Model(&dao.RefCase{}).Select("id").Where("contribution_id = ?", 1).Scan(&ids).Error
				require.NoError(t, err)
				assert.Equal(t, 0, len(ids))
			},
			wantCode: 200,
		},
	}
	for _, tc := range testcases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/resume/project/contribution/del", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[int64]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			tc.after(t)
			// 清理数据
			err = s.db.Exec("TRUNCATE  TABLE `resume_projects`").Error
			require.NoError(s.T(), err)
			err = s.db.Exec("TRUNCATE TABLE `contributions`").Error
			require.NoError(s.T(), err)
			err = s.db.Exec("TRUNCATE  TABLE `difficulties`").Error
			require.NoError(s.T(), err)
			err = s.db.Exec("TRUNCATE  TABLE `ref_cases`").Error
			require.NoError(s.T(), err)
		})
	}
}

func (s *ProjectTestSuite) TestResumeInfo() {
	testcases := []struct {
		name     string
		before   func(t *testing.T)
		req      web.IDItem
		wantResp test.Result[web.Project]
		wantCode int
	}{
		{
			name:     "获取某个项目的详情",
			wantCode: 200,
			req: web.IDItem{
				ID: 1,
			},
			before: func(t *testing.T) {
				_, err := s.pdao.Upsert(context.Background(), dao.ResumeProject{
					ID:           1,
					StartTime:    2,
					EndTime:      666,
					Uid:          uid,
					Name:         "projectnew",
					Introduction: "introductionnew",
					Core:         true,
				})
				require.NoError(t, err)
				err = s.pdao.SaveDifficulty(context.Background(), dao.Difficulty{
					ID:        1,
					ProjectID: 1,
					CaseID:    2,
					Level:     1,
					Desc:      "desc",
				})
				require.NoError(t, err)
				err = s.pdao.SaveDifficulty(context.Background(), dao.Difficulty{
					ID:        2,
					ProjectID: 1,
					CaseID:    3,
					Level:     1,
					Desc:      "diff_desc",
				})
				require.NoError(t, err)
				_, err = s.pdao.SaveContribution(context.Background(), dao.Contribution{
					ID:        1,
					Type:      "type",
					ProjectID: 1,
					Desc:      "desc",
				}, []dao.RefCase{
					{
						ID:             1,
						CaseID:         2,
						Highlight:      true,
						ContributionID: 1,
						Level:          1,
					},
					{
						ID:             2,
						CaseID:         3,
						Highlight:      false,
						ContributionID: 1,
						Level:          2,
					},
				})
				require.NoError(t, err)
			},
			wantResp: test.Result[web.Project]{
				Data: web.Project{
					Id:           1,
					StartTime:    2,
					EndTime:      666,
					Uid:          uid,
					Name:         "projectnew",
					Introduction: "introductionnew",
					Core:         true,
					Difficulties: []web.Difficulty{
						{
							ID:   1,
							Desc: "desc",
							Case: web.Case{
								Id:            2,
								ExamineResult: 2 % 4,
								Title:         "这是案例2",
								Introduction:  "这是案例的简介2",
								Level:         1,
							},
						},
						{
							ID:   2,
							Desc: "diff_desc",
							Case: web.Case{
								Id:            3,
								ExamineResult: 3 % 4,
								Level:         1,
								Title:         "这是案例3",
								Introduction:  "这是案例的简介3",
							},
						},
					},
					Contributions: []web.Contribution{
						{
							ID:   1,
							Type: "type",
							Desc: "desc",
							RefCases: []web.Case{
								{
									Id:            2,
									ExamineResult: 2 % 4,
									Highlight:     true,
									Level:         1,
									Title:         "这是案例2",
									Introduction:  "这是案例的简介2",
								},
								{
									Id:            3,
									ExamineResult: 3 % 4,
									Highlight:     false,
									Level:         2,
									Title:         "这是案例3",
									Introduction:  "这是案例的简介3",
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tc := range testcases {
		tc := tc
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/resume/project/info", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.Project]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			require.Equal(t, tc.wantResp, recorder.MustScan())
			// 清理数据
			err = s.db.Exec("TRUNCATE  TABLE `resume_projects`").Error
			require.NoError(s.T(), err)
			err = s.db.Exec("TRUNCATE TABLE `contributions`").Error
			require.NoError(s.T(), err)
			err = s.db.Exec("TRUNCATE  TABLE `difficulties`").Error
			require.NoError(s.T(), err)
			err = s.db.Exec("TRUNCATE  TABLE `ref_cases`").Error
			require.NoError(s.T(), err)
		})
	}
}

func (s *ProjectTestSuite) TestResumeList() {
	for i := 1; i < 4; i++ {
		_, err := s.pdao.Upsert(context.Background(), dao.ResumeProject{
			ID:           int64(i),
			StartTime:    int64(i),
			EndTime:      int64(i + 1000),
			Uid:          uid,
			Introduction: "introduction",
			Name:         fmt.Sprintf("项目 %d", i),
			Core:         i%2 == 1,
		})
		require.NoError(s.T(), err)
		err = s.pdao.SaveDifficulty(context.Background(), dao.Difficulty{
			ID:        int64(i*10 + i),
			ProjectID: int64(i),
			CaseID:    int64(i*10 + i),
			Level:     1,
			Desc:      fmt.Sprintf("desc_%d", i*10+i),
		})
		require.NoError(s.T(), err)
		err = s.pdao.SaveDifficulty(context.Background(), dao.Difficulty{
			ID:        int64(i*10 + i + 1),
			ProjectID: int64(i),
			CaseID:    int64(i*10 + i + 1),
			Level:     1,
			Desc:      fmt.Sprintf("desc_%d", i*10+i+1),
		})
		require.NoError(s.T(), err)
		_, err = s.pdao.SaveContribution(context.Background(), dao.Contribution{
			ID:        int64(i*20 + i),
			Type:      "type",
			ProjectID: int64(i),
			Desc:      "desc",
		}, []dao.RefCase{
			{
				ID:             int64(i*20 + i + 1),
				CaseID:         int64(i*20 + i + 1),
				Highlight:      true,
				ContributionID: int64(i*20 + i),
				Level:          1,
			},
			{
				ID:             int64(i*20 + i + 2),
				CaseID:         int64(i*20 + i + 2),
				Highlight:      false,
				ContributionID: int64(i*20 + i),
				Level:          2,
			},
		})

	}
	_, err := s.pdao.Upsert(context.Background(), dao.ResumeProject{
		ID:        int64(5),
		StartTime: int64(5),
		EndTime:   int64(5 + 1000),
		Uid:       456,
		Name:      fmt.Sprintf("项目 %d", 5),
		Core:      true,
	})
	req, err := http.NewRequest(http.MethodPost,
		"/resume/project/list", iox.NewJSONReader(nil))
	require.NoError(s.T(), err)
	req.Header.Set("content-type", "application/json")
	recorder := test.NewJSONResponseRecorder[[]web.Project]()
	s.server.ServeHTTP(recorder, req)

	require.Equal(s.T(), 200, recorder.Code)
	data := recorder.MustScan().Data
	assert.Equal(s.T(), []web.Project{
		{
			Id:           3,
			StartTime:    3,
			EndTime:      1003,
			Uid:          uid,
			Introduction: "introduction",
			Name:         fmt.Sprintf("项目 %d", 3),
			Core:         true,
			Contributions: []web.Contribution{
				{
					ID:   63,
					Type: "type",
					Desc: "desc",
					RefCases: []web.Case{
						{
							Id:            64,
							Title:         "这是案例64",
							Introduction:  "这是案例的简介64",
							ExamineResult: 64 % 4,
							Highlight:     true,
							Level:         1,
						},
						{
							Id:            65,
							Title:         "这是案例65",
							Introduction:  "这是案例的简介65",
							ExamineResult: 65 % 4,
							Highlight:     false,
							Level:         2,
						},
					},
				},
			},
			Difficulties: []web.Difficulty{
				{
					ID:   33,
					Desc: "desc_33",
					Case: web.Case{
						Id:            33,
						Title:         "这是案例33",
						Introduction:  "这是案例的简介33",
						ExamineResult: 33 % 4,
						Highlight:     false,
						Level:         1,
					},
				},
				{
					ID:   34,
					Desc: "desc_34",
					Case: web.Case{
						Id:            34,
						Title:         "这是案例34",
						Introduction:  "这是案例的简介34",
						ExamineResult: 34 % 4,
						Highlight:     false,
						Level:         1,
					},
				},
			},
		},
		{
			Id:           2,
			StartTime:    2,
			EndTime:      1002,
			Introduction: "introduction",
			Uid:          uid,
			Name:         fmt.Sprintf("项目 %d", 2),
			Core:         false,
			Contributions: []web.Contribution{
				{
					ID:   42,
					Type: "type",
					Desc: "desc",
					RefCases: []web.Case{
						{
							Id:            43,
							Title:         "这是案例43",
							Introduction:  "这是案例的简介43",
							ExamineResult: 43 % 4,
							Highlight:     true,
							Level:         1,
						},
						{
							Id:            44,
							Title:         "这是案例44",
							Introduction:  "这是案例的简介44",
							ExamineResult: 44 % 4,
							Highlight:     false,
							Level:         2,
						},
					},
				},
			},
			Difficulties: []web.Difficulty{
				{
					ID:   22,
					Desc: "desc_22",
					Case: web.Case{
						Id:            22,
						Title:         "这是案例22",
						Introduction:  "这是案例的简介22",
						ExamineResult: 22 % 4,
						Level:         1,
					},
				},
				{
					ID:   23,
					Desc: "desc_23",
					Case: web.Case{
						Id:            23,
						Title:         "这是案例23",
						Introduction:  "这是案例的简介23",
						ExamineResult: 23 % 4,
						Level:         1,
					},
				},
			},
		},
		{
			Id:           1,
			StartTime:    1,
			EndTime:      1001,
			Introduction: "introduction",
			Uid:          uid,
			Name:         fmt.Sprintf("项目 %d", 1),
			Core:         true,
			Contributions: []web.Contribution{
				{
					ID:   21,
					Type: "type",
					Desc: "desc",
					RefCases: []web.Case{
						{
							Id:            22,
							Title:         "这是案例22",
							Introduction:  "这是案例的简介22",
							ExamineResult: 22 % 4,
							Highlight:     true,
							Level:         1,
						},
						{
							Id:            23,
							Title:         "这是案例23",
							Introduction:  "这是案例的简介23",
							ExamineResult: 23 % 4,
							Highlight:     false,
							Level:         2,
						},
					},
				},
			},
			Difficulties: []web.Difficulty{
				{
					ID:   11,
					Desc: "desc_11",
					Case: web.Case{
						Id:            11,
						Title:         "这是案例11",
						Introduction:  "这是案例的简介11",
						ExamineResult: 11 % 4,
						Level:         1,
					},
				},
				{
					ID:   12,
					Desc: "desc_12",
					Case: web.Case{
						Id:            12,
						Title:         "这是案例12",
						Introduction:  "这是案例的简介12",
						ExamineResult: 12 % 4,
						Level:         1,
					},
				},
			},
		},
	}, data)
	// 清理数据
	err = s.db.Exec("TRUNCATE  TABLE `resume_projects`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE `contributions`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE  TABLE `difficulties`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE  TABLE `ref_cases`").Error
	require.NoError(s.T(), err)

}

func (s *ProjectTestSuite) assertContribution(actual *dao.Contribution,
	expected *dao.Contribution) {
	t := s.T()
	require.True(t, actual.Ctime != 0)
	require.True(t, actual.Utime != 0)
	actual.Ctime = 0
	actual.Utime = 0
	assert.Equal(t, expected, actual)
}
