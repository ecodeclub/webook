//go:build e2e

package integration

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/cases"
	casemocks "github.com/ecodeclub/webook/internal/cases/mocks"
	baguwen "github.com/ecodeclub/webook/internal/question"
	quemocks "github.com/ecodeclub/webook/internal/question/mocks"
	"go.uber.org/mock/gomock"

	"github.com/ecodeclub/ekit/iox"
	"github.com/ecodeclub/ekit/sqlx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/skill/internal/integration/startup"
	"github.com/ecodeclub/webook/internal/skill/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/skill/internal/web"
	"github.com/ecodeclub/webook/internal/test"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/ego-component/egorm"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/econf"
	"github.com/gotomicro/ego/server/egin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const uid = 2061

type HandlerTestSuite struct {
	suite.Suite
	server *egin.Component
	db     *egorm.Component
	dao    dao.SkillDAO
}

func (s *HandlerTestSuite) TearDownTest() {
	err := s.db.Exec("TRUNCATE  TABLE `skill`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE `skill_level`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE  TABLE `skill_refs`").Error
	require.NoError(s.T(), err)
}

func (s *HandlerTestSuite) SetupSuite() {
	ctrl := gomock.NewController(s.T())
	queSvc := quemocks.NewMockService(ctrl)
	queSvc.EXPECT().GetPubByIDs(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, ids []int64) ([]baguwen.Question, error) {
			return slice.Map(ids, func(idx int, src int64) baguwen.Question {
				return baguwen.Question{
					Id:    src,
					Title: "这是问题" + strconv.FormatInt(src, 10),
				}
			}), nil
		}).AnyTimes()

	caseSvc := casemocks.NewMockService(ctrl)
	caseSvc.EXPECT().GetPubByIDs(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, ids []int64) ([]cases.Case, error) {
			return slice.Map(ids, func(idx int, src int64) cases.Case {
				return cases.Case{
					Id:    src,
					Title: "这是案例" + strconv.FormatInt(src, 10),
				}
			}), nil
		}).AnyTimes()

	handler, err := startup.InitHandler(
		&baguwen.Module{Svc: queSvc},
		&cases.Module{Svc: caseSvc},
	)
	require.NoError(s.T(), err)
	econf.Set("server", map[string]any{"contextTimeout": "1s"})
	server := egin.Load("server").Build()
	server.Use(func(ctx *gin.Context) {
		ctx.Set("_session", session.NewMemorySession(session.Claims{
			Uid:  uid,
			Data: map[string]string{"creator": "true"},
		}))
	})
	handler.PrivateRoutes(server.Engine)
	s.server = server
	s.db = testioc.InitDB()
	err = dao.InitTables(s.db)
	require.NoError(s.T(), err)
	s.dao = dao.NewSkillDAO(s.db)
}

func (s *HandlerTestSuite) TestSave() {
	testCases := []struct {
		name     string
		before   func(t *testing.T)
		after    func(t *testing.T)
		req      web.SaveReq
		wantCode int
		wantResp test.Result[int64]
	}{
		{
			name: "新增",
			before: func(t *testing.T) {

			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				skill, err := s.dao.Info(ctx, 1)
				require.NoError(t, err)
				skillLevels, err := s.dao.SkillLevelInfo(ctx, 1)
				require.NoError(t, err)
				s.assertSkill(dao.Skill{
					Labels: sqlx.JsonColumn[[]string]{
						Val:   []string{"mysql"},
						Valid: true,
					},
					Name: "mysql",
					Desc: "mysql_desc",
				}, skill)
				wantLevels := []dao.SkillLevel{
					{
						Sid:   1,
						Level: "basic",
						Desc:  "mysql_basic",
					},
					{
						Sid:   1,
						Level: "intermediate",
						Desc:  "mysql_intermediate",
					},
					{
						Sid:   1,
						Level: "advanced",
						Desc:  "mysql_advanced",
					},
				}
				assert.Equal(t, len(wantLevels), len(skillLevels))
				for idx := range skillLevels {
					current := &(skillLevels[idx])
					assert.True(t, current.Id > 0)
					assert.True(t, current.Utime > 0)
					assert.True(t, current.Ctime > 0)
					current.Id = 0
					current.Ctime = 0
					current.Utime = 0
				}
				assert.ElementsMatch(t, wantLevels, skillLevels)
			},
			req: web.SaveReq{
				Skill: web.Skill{
					Labels: []string{"mysql"},
					Name:   "mysql",
					Desc:   "mysql_desc",
					Basic: web.SkillLevel{
						Desc: "mysql_basic",
					},
					Intermediate: web.SkillLevel{
						Desc: "mysql_intermediate",
					},
					Advanced: web.SkillLevel{
						Desc: "mysql_advanced",
					},
				},
			},
			wantCode: 200,
			wantResp: test.Result[int64]{
				Data: 1,
			},
		},
		{
			name: "更新",
			before: func(t *testing.T) {
				err := s.db.Create(&dao.Skill{
					Id: 2,
					Labels: sqlx.JsonColumn[[]string]{
						Val:   []string{"old_mysql"},
						Valid: true,
					},
					Name:  "old_mysql",
					Desc:  "old_mysql_desc",
					Ctime: time.Now().UnixMilli(),
					Utime: time.Now().UnixMilli(),
				}).Error
				require.NoError(t, err)
				err = s.db.Create([]*dao.SkillLevel{
					{
						Sid:   2,
						Level: "old_mysql_level1",
						Desc:  "old_mysql_desc",
						Ctime: time.Now().UnixMilli(),
						Utime: time.Now().UnixMilli(),
					},
					{
						Sid:   2,
						Level: "old_mysql_level2",
						Desc:  "old_mysql_desc",
						Ctime: time.Now().UnixMilli(),
						Utime: time.Now().UnixMilli(),
					},
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				skill, err := s.dao.Info(ctx, 2)
				require.NoError(t, err)
				skillLevels, err := s.dao.SkillLevelInfo(ctx, 2)
				require.NoError(t, err)
				s.assertSkill(dao.Skill{
					Labels: sqlx.JsonColumn[[]string]{
						Val:   []string{"mysql"},
						Valid: true,
					},
					Name: "mysql",
					Desc: "mysql_desc",
				}, skill)
				wantLevels := []dao.SkillLevel{
					{
						Sid:   2,
						Level: "basic",
						Desc:  "mysql_desc",
					},
					{
						Sid:   2,
						Level: "intermediate",
						Desc:  "mysql_desc",
					},
					{
						Sid:   2,
						Level: "advanced",
						Desc:  "mysql_desc",
					},
				}
				assert.Equal(t, len(wantLevels), len(skillLevels))
				for idx := range skillLevels {
					current := &(skillLevels[idx])
					assert.True(t, current.Id > 0)
					assert.True(t, current.Utime > 0)
					assert.True(t, current.Ctime > 0)
					current.Id = 0
					current.Ctime = 0
					current.Utime = 0
				}
				assert.ElementsMatch(t, wantLevels, skillLevels)
			},
			req: web.SaveReq{
				Skill: web.Skill{
					ID:     2,
					Labels: []string{"mysql"},
					Name:   "mysql",
					Desc:   "mysql_desc",
					Basic: web.SkillLevel{
						Id:   1,
						Desc: "mysql_desc",
					},
					Intermediate: web.SkillLevel{
						Id:   2,
						Desc: "mysql_desc",
					},
					Advanced: web.SkillLevel{
						Desc: "mysql_desc",
					},
				},
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
				"/skill/save", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[int64]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
			tc.after(t)
			// 清理数据
			err = s.db.Exec("TRUNCATE  TABLE `skill`").Error
			require.NoError(s.T(), err)
			err = s.db.Exec("TRUNCATE TABLE `skill_level`").Error
			require.NoError(s.T(), err)
			err = s.db.Exec("TRUNCATE  TABLE `skill_refs`").Error
			require.NoError(s.T(), err)
		})
	}
}

func (s *HandlerTestSuite) TestSaveRefs() {
	testCases := []struct {
		name     string
		before   func(t *testing.T)
		after    func(t *testing.T)
		req      web.SaveReq
		wantCode int
	}{
		{
			name: "新建",
			before: func(t *testing.T) {
				err := s.db.Create(&dao.Skill{
					Id: 1,
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				refs, err := s.dao.Refs(ctx, 1)
				require.NoError(t, err)
				wantReqs := []dao.SkillRef{
					{
						Sid:   1,
						Slid:  1,
						Rid:   12,
						Rtype: "question",
					},
					{
						Sid:   1,
						Slid:  1,
						Rid:   23,
						Rtype: "question",
					},
					{
						Sid:   1,
						Slid:  2,
						Rtype: "question",
						Rid:   34,
					},
					{
						Sid:   1,
						Slid:  2,
						Rid:   45,
						Rtype: "case",
					},
					{
						Sid:   1,
						Slid:  3,
						Rid:   67,
						Rtype: "case",
					},
				}
				for idx := range refs {
					ref := &(refs[idx])
					assert.True(t, ref.Id > 0)
					assert.True(t, ref.Ctime > 0)
					assert.True(t, ref.Utime > 0)
					ref.Id = 0
					ref.Ctime = 0
					ref.Utime = 0
				}
				assert.ElementsMatch(t, wantReqs, refs)
			},
			req: web.SaveReq{
				Skill: web.Skill{
					ID: 1,
					Basic: web.SkillLevel{
						Id:   1,
						Desc: "这是 basic ",
						Questions: []web.Question{
							{Id: 12},
							{Id: 23},
						},
					},
					Intermediate: web.SkillLevel{
						Id: 2,
						Questions: []web.Question{
							{Id: 34},
						},
						Cases: []web.Case{
							{Id: 45},
						},
					},
					Advanced: web.SkillLevel{
						Id: 3,
						Cases: []web.Case{
							{Id: 67},
						},
					},
				},
			},
			wantCode: 200,
		},
		{
			name: "更新",
			before: func(t *testing.T) {
				err := s.db.Create([]*dao.SkillRef{
					{
						Id:    1,
						Sid:   1,
						Slid:  1,
						Rtype: "case",
						Rid:   1,
					},
					{
						Id:    2,
						Sid:   1,
						Slid:  1,
						Rid:   1,
						Rtype: "question",
					},
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				refs, err := s.dao.Refs(ctx, 1)
				require.NoError(t, err)
				wantReqs := []dao.SkillRef{
					{
						Sid:   1,
						Slid:  1,
						Rid:   12,
						Rtype: "question",
					},
					{
						Sid:   1,
						Slid:  1,
						Rid:   23,
						Rtype: "question",
					},
					{
						Sid:   1,
						Slid:  2,
						Rtype: "question",
						Rid:   34,
					},
					{
						Sid:   1,
						Slid:  2,
						Rid:   45,
						Rtype: "case",
					},
					{
						Sid:   1,
						Slid:  3,
						Rid:   67,
						Rtype: "case",
					},
				}
				for idx := range refs {
					ref := &(refs[idx])
					assert.True(t, ref.Id > 0)
					assert.True(t, ref.Ctime > 0)
					assert.True(t, ref.Utime > 0)
					ref.Id = 0
					ref.Ctime = 0
					ref.Utime = 0
				}
				assert.ElementsMatch(t, wantReqs, refs)
			},
			req: web.SaveReq{
				Skill: web.Skill{
					ID: 1,
					Basic: web.SkillLevel{
						Id:   1,
						Desc: "这是 basic ",
						Questions: []web.Question{
							{Id: 12},
							{Id: 23},
						},
					},
					Intermediate: web.SkillLevel{
						Id: 2,
						Questions: []web.Question{
							{Id: 34},
						},
						Cases: []web.Case{
							{Id: 45},
						},
					},
					Advanced: web.SkillLevel{
						Id: 3,
						Cases: []web.Case{
							{Id: 67},
						},
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
				"/skill/save-refs", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[int64]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			tc.after(t)
			// 清理数据
			err = s.db.Exec("TRUNCATE  TABLE `skill`").Error
			require.NoError(s.T(), err)
			err = s.db.Exec("TRUNCATE TABLE `skill_level`").Error
			require.NoError(s.T(), err)
			err = s.db.Exec("TRUNCATE  TABLE `skill_refs`").Error
			require.NoError(s.T(), err)
		})
	}
}

func (s *HandlerTestSuite) TestDetail() {
	t := s.T()
	err := s.db.Create(&dao.Skill{
		Id: 2,
		Labels: sqlx.JsonColumn[[]string]{
			Val:   []string{"mysql"},
			Valid: true,
		},
		Name:  "mysql",
		Desc:  "mysql_desc",
		Ctime: time.Now().UnixMilli(),
		Utime: time.Now().UnixMilli(),
	}).Error
	require.NoError(t, err)
	err = s.db.Create([]*dao.SkillLevel{
		{
			Id:    1,
			Sid:   2,
			Level: "basic",
			Desc:  "mysql_desc_basic",
			Ctime: time.Now().UnixMilli(),
			Utime: time.Now().UnixMilli(),
		},
		{
			Id:    2,
			Sid:   2,
			Level: "intermediate",
			Desc:  "mysql_desc_inter",
			Ctime: time.Now().UnixMilli(),
			Utime: time.Now().UnixMilli(),
		},
	}).Error
	require.NoError(t, err)
	s.db.Create([]*dao.SkillRef{
		{
			Id:    1,
			Slid:  1,
			Sid:   2,
			Rtype: "case",
			Rid:   1,
			Ctime: time.Now().UnixMilli(),
			Utime: time.Now().UnixMilli(),
		},
		{
			Id:    2,
			Slid:  1,
			Sid:   2,
			Rtype: "question",
			Rid:   2,
			Ctime: time.Now().UnixMilli(),
			Utime: time.Now().UnixMilli(),
		},
		{
			Id:    3,
			Slid:  2,
			Sid:   2,
			Rtype: "question",
			Rid:   1,
			Ctime: time.Now().UnixMilli(),
			Utime: time.Now().UnixMilli(),
		},
	})
	sid := web.Sid{
		Sid: 2,
	}
	req, err := http.NewRequest(http.MethodPost,
		"/skill/detail", iox.NewJSONReader(sid))
	req.Header.Set("content-type", "application/json")
	require.NoError(t, err)
	recorder := test.NewJSONResponseRecorder[web.Skill]()
	s.server.ServeHTTP(recorder, req)
	require.Equal(t, 200, recorder.Code)
	resp := recorder.MustScan().Data
	assert.True(t, len(resp.Utime) > 0)
	resp.Utime = ""
	assert.Equal(t, web.Skill{
		ID: 2,
		Labels: []string{
			"mysql",
		},
		Name: "mysql",
		Desc: "mysql_desc",
		Basic: web.SkillLevel{
			Id:   1,
			Desc: "mysql_desc_basic",
			Questions: []web.Question{
				{Id: 2},
			},
			Cases: []web.Case{
				{Id: 1},
			},
		},
		Intermediate: web.SkillLevel{
			Id:   2,
			Desc: "mysql_desc_inter",
			Questions: []web.Question{
				{Id: 1},
			},
			Cases: []web.Case{},
		},
		Advanced: web.SkillLevel{
			Questions: []web.Question{},
			Cases:     []web.Case{},
		},
	}, resp)
}

func (s *HandlerTestSuite) TestDetailRef() {
	t := s.T()
	err := s.db.Create(&dao.Skill{
		Id: 2,
		Labels: sqlx.JsonColumn[[]string]{
			Val:   []string{"mysql"},
			Valid: true,
		},
		Name:  "mysql",
		Desc:  "mysql_desc",
		Ctime: time.Now().UnixMilli(),
		Utime: time.Now().UnixMilli(),
	}).Error
	require.NoError(t, err)
	err = s.db.Create([]*dao.SkillLevel{
		{
			Id:    1,
			Sid:   2,
			Level: "basic",
			Desc:  "mysql_desc_basic",
			Ctime: time.Now().UnixMilli(),
			Utime: time.Now().UnixMilli(),
		},
		{
			Id:    2,
			Sid:   2,
			Level: "intermediate",
			Desc:  "mysql_desc_inter",
			Ctime: time.Now().UnixMilli(),
			Utime: time.Now().UnixMilli(),
		},
	}).Error
	require.NoError(t, err)
	s.db.Create([]*dao.SkillRef{
		{
			Id:    1,
			Slid:  1,
			Sid:   2,
			Rtype: "case",
			Rid:   1,
			Ctime: time.Now().UnixMilli(),
			Utime: time.Now().UnixMilli(),
		},
		{
			Id:    2,
			Slid:  1,
			Sid:   2,
			Rtype: "question",
			Rid:   2,
			Ctime: time.Now().UnixMilli(),
			Utime: time.Now().UnixMilli(),
		},
		{
			Id:    3,
			Slid:  2,
			Sid:   2,
			Rtype: "question",
			Rid:   1,
			Ctime: time.Now().UnixMilli(),
			Utime: time.Now().UnixMilli(),
		},
	})
	sid := web.Sid{
		Sid: 2,
	}
	req, err := http.NewRequest(http.MethodPost,
		"/skill/detail-refs", iox.NewJSONReader(sid))
	req.Header.Set("content-type", "application/json")
	require.NoError(t, err)
	recorder := test.NewJSONResponseRecorder[web.Skill]()
	s.server.ServeHTTP(recorder, req)
	require.Equal(t, 200, recorder.Code)
	resp := recorder.MustScan().Data
	assert.True(t, len(resp.Utime) > 0)
	resp.Utime = ""
	assert.Equal(t, web.Skill{
		ID: 2,
		Labels: []string{
			"mysql",
		},
		Name: "mysql",
		Desc: "mysql_desc",
		Basic: web.SkillLevel{
			Id:   1,
			Desc: "mysql_desc_basic",
			Questions: []web.Question{
				{Id: 2, Title: "这是问题2"},
			},
			Cases: []web.Case{
				{Id: 1, Title: "这是案例1"},
			},
		},
		Intermediate: web.SkillLevel{
			Id:   2,
			Desc: "mysql_desc_inter",
			Questions: []web.Question{
				{Id: 1, Title: "这是问题1"},
			},
			Cases: []web.Case{},
		},
		Advanced: web.SkillLevel{
			Questions: []web.Question{},
			Cases:     []web.Case{},
		},
	}, resp)
}

func (s *HandlerTestSuite) TestList() {
	skills := make([]*dao.Skill, 0, 100)
	for i := 1; i <= 100; i++ {
		name := fmt.Sprintf("mysql%d", i)
		skills = append(skills, &dao.Skill{
			Id: int64(i),
			Labels: sqlx.JsonColumn[[]string]{
				Val:   []string{name},
				Valid: true,
			},
			Name:  name,
			Desc:  fmt.Sprintf("%s_desc", name),
			Ctime: time.Unix(0, 0).UnixMilli(),
			Utime: time.Unix(0, 0).UnixMilli(),
		})
	}
	err := s.db.Create(&skills).Error
	require.NoError(s.T(), err)
	testCases := []struct {
		name     string
		req      web.Page
		wantCode int
		wantResp test.Result[web.SkillList]
	}{
		{
			name: "获取全部",
			req: web.Page{
				Limit:  2,
				Offset: 0,
			},
			wantCode: 200,
			wantResp: test.Result[web.SkillList]{
				Data: web.SkillList{
					Total: 100,
					Skills: []web.Skill{
						{
							ID:   100,
							Name: "mysql100",
							Desc: "mysql100_desc",
							Labels: []string{
								"mysql100",
							},
							Utime: time.Unix(0, 0).Format(time.DateTime),
							Basic: web.SkillLevel{
								Questions: []web.Question{},
								Cases:     []web.Case{},
							},
							Intermediate: web.SkillLevel{
								Questions: []web.Question{},
								Cases:     []web.Case{},
							},
							Advanced: web.SkillLevel{
								Questions: []web.Question{},
								Cases:     []web.Case{},
							},
						},
						{
							ID:   99,
							Name: "mysql99",
							Desc: "mysql99_desc",
							Labels: []string{
								"mysql99",
							},
							Utime: time.Unix(0, 0).Format(time.DateTime),
							Basic: web.SkillLevel{
								Questions: []web.Question{},
								Cases:     []web.Case{},
							},
							Intermediate: web.SkillLevel{
								Questions: []web.Question{},
								Cases:     []web.Case{},
							},
							Advanced: web.SkillLevel{
								Questions: []web.Question{},
								Cases:     []web.Case{},
							},
						},
					},
				},
			},
		},
		{
			name: "部分获取",
			req: web.Page{
				Limit:  2,
				Offset: 99,
			},
			wantCode: 200,
			wantResp: test.Result[web.SkillList]{
				Data: web.SkillList{
					Total: 100,
					Skills: []web.Skill{
						{
							ID:   1,
							Name: "mysql1",
							Desc: "mysql1_desc",
							Labels: []string{
								"mysql1",
							},
							Utime: time.Unix(0, 0).Format(time.DateTime),
							Basic: web.SkillLevel{
								Questions: []web.Question{},
								Cases:     []web.Case{},
							},
							Intermediate: web.SkillLevel{
								Questions: []web.Question{},
								Cases:     []web.Case{},
							},
							Advanced: web.SkillLevel{
								Questions: []web.Question{},
								Cases:     []web.Case{},
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
				"/skill/list", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.SkillList]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
		})
	}

}

func (s *HandlerTestSuite) TestRefsByLevelIDs() {
	err := s.db.Create([]*dao.SkillRef{
		{
			Id:    1,
			Slid:  1,
			Sid:   2,
			Rtype: "case",
			Rid:   1,
			Ctime: time.Now().UnixMilli(),
			Utime: time.Now().UnixMilli(),
		},
		{
			Id:    2,
			Slid:  1,
			Sid:   2,
			Rtype: "question",
			Rid:   2,
			Ctime: time.Now().UnixMilli(),
			Utime: time.Now().UnixMilli(),
		},
		{
			Id:    3,
			Slid:  2,
			Sid:   2,
			Rtype: "question",
			Rid:   1,
			Ctime: time.Now().UnixMilli(),
			Utime: time.Now().UnixMilli(),
		},
	}).Error
	require.NoError(s.T(), err)
	testCases := []struct {
		name string
		req  web.IDs

		wantCode int
		wantResp test.Result[[]web.SkillLevel]
	}{
		{
			name: "查询成功",
			req: web.IDs{
				IDs: []int64{1, 2},
			},
			wantCode: 200,
			wantResp: test.Result[[]web.SkillLevel]{
				Data: []web.SkillLevel{
					{
						Id: 1,
						Questions: []web.Question{
							{Id: 2, Title: "这是问题2"},
						},
						Cases: []web.Case{
							{Id: 1, Title: "这是案例1"},
						},
					},
					{
						Id: 2,
						Questions: []web.Question{
							{Id: 1, Title: "这是问题1"},
						},
						Cases: []web.Case{},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost,
				"/skill/level-refs", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[[]web.SkillLevel]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
		})
	}

}

func (s *HandlerTestSuite) assertSkill(wantSKill dao.Skill, actualSkill dao.Skill) {
	t := s.T()
	require.True(t, actualSkill.Id > 0)
	require.True(t, actualSkill.Utime > 0)
	require.True(t, actualSkill.Ctime > 0)
	actualSkill.Id = 0
	actualSkill.Utime = 0
	actualSkill.Ctime = 0
	assert.Equal(t, wantSKill, actualSkill)
}

func TestHandler(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}
