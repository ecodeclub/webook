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

func (s *HandlerTestSuite) TearDownSuite() {
	err := s.db.Exec("DROP TABLE `skill`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("DROP TABLE `skill_level`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("DROP TABLE `skill_pre_request`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("DROP TABLE `pub_skill`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("DROP TABLE `pub_skill_level`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("DROP TABLE `pub_skill_pre_request`").Error
	require.NoError(s.T(), err)
}

func (s *HandlerTestSuite) TearDownTest() {
	err := s.db.Exec("TRUNCATE  TABLE `skill`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE `skill_level`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE  TABLE `skill_pre_request`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE  TABLE `pub_skill`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE `pub_skill_level`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE  TABLE `pub_skill_pre_request`").Error
	require.NoError(s.T(), err)
}
func (s *HandlerTestSuite) SetupSuite() {
	handler, err := startup.InitHandler()
	require.NoError(s.T(), err)
	econf.Set("server", map[string]any{"contextTimeout": "1s"})
	server := egin.Load("server").Build()
	server.Use(func(ctx *gin.Context) {
		ctx.Set("_session", session.NewMemorySession(session.Claims{
			Uid:  uid,
			Data: map[string]string{"admin": "true"},
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
					s.assertSkillLevel(wantLevels[idx], skillLevels[idx])
				}
			},
			req: web.SaveReq{
				Skill: web.Skill{
					Labels: []string{"mysql"},
					Name:   "mysql",
					Desc:   "mysql_desc",
					Levels: []web.SkillLevel{
						{
							Level: "basic",
							Desc:  "mysql_basic",
						},
						{
							Level: "intermediate",
							Desc:  "mysql_intermediate",
						},
						{
							Level: "advanced",
							Desc:  "mysql_advanced",
						},
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
					Name: "old_mysql",
					Desc: "old_mysql_desc",
					Base: dao.Base{
						Ctime: time.Now().UnixMilli(),
						Utime: time.Now().UnixMilli(),
					},
				}).Error
				require.NoError(t, err)
				s.db.Create([]*dao.SkillLevel{
					{
						Sid:   2,
						Level: "old_mysql_level1",
						Desc:  "old_mysql_desc",
						Base: dao.Base{
							Ctime: time.Now().UnixMilli(),
							Utime: time.Now().UnixMilli(),
						},
					},
					{
						Sid:   2,
						Level: "old_mysql_level2",
						Desc:  "old_mysql_desc",
						Base: dao.Base{
							Ctime: time.Now().UnixMilli(),
							Utime: time.Now().UnixMilli(),
						},
					},
				},
				)
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
						Level: "mysql_level1",
						Desc:  "mysql_desc",
					},
					{
						Sid:   2,
						Level: "mysql_level2",
						Desc:  "mysql_desc",
					},
					{
						Sid:   2,
						Level: "mysql_level3",
						Desc:  "mysql_desc",
					},
				}
				assert.Equal(t, len(wantLevels), len(skillLevels))
				for idx := range skillLevels {
					s.assertSkillLevel(wantLevels[idx], skillLevels[idx])
				}
			},
			req: web.SaveReq{
				Skill: web.Skill{
					ID:     2,
					Labels: []string{"mysql"},
					Name:   "mysql",
					Desc:   "mysql_desc",
					Levels: []web.SkillLevel{
						{
							Id:    1,
							Level: "mysql_level1",
							Desc:  "mysql_desc",
						},
						{
							Id:    2,
							Level: "mysql_level2",
							Desc:  "mysql_desc",
						},
						{
							Level: "mysql_level3",
							Desc:  "mysql_desc",
						},
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
			err = s.db.Exec("TRUNCATE  TABLE `skill_pre_request`").Error
			require.NoError(s.T(), err)
		})
	}
}

func (s *HandlerTestSuite) TestSaveReq() {
	testCases := []struct {
		name     string
		before   func(t *testing.T)
		after    func(t *testing.T)
		req      web.SaveRequestReq
		wantCode int
	}{
		{
			name: "新建",
			before: func(t *testing.T) {
				err := s.db.Create(&dao.Skill{
					Id: 1,
					Labels: sqlx.JsonColumn[[]string]{
						Val:   []string{"old_mysql"},
						Valid: true,
					},
					Name: "redis",
					Desc: "redis_desc",
					Base: dao.Base{
						Ctime: time.Now().UnixMilli(),
						Utime: time.Now().UnixMilli(),
					},
				}).Error
				require.NoError(t, err)
				s.db.Create([]*dao.SkillLevel{
					{
						Id:    1,
						Sid:   1,
						Level: "redis_level1",
						Desc:  "redis_desc",
						Base: dao.Base{
							Ctime: time.Now().UnixMilli(),
							Utime: time.Now().UnixMilli(),
						},
					},
					{
						Id:    2,
						Sid:   1,
						Level: "redis_level2",
						Desc:  "redis_desc",
						Base: dao.Base{
							Ctime: time.Now().UnixMilli(),
							Utime: time.Now().UnixMilli(),
						},
					},
				},
				)
			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				reqs, err := s.dao.RequestInfo(ctx, 1)
				require.NoError(t, err)
				wantReqs := []dao.SkillPreRequest{
					{
						Sid:   1,
						Slid:  1,
						Rid:   1,
						Rtype: "case",
					},
					{
						Sid:   1,
						Slid:  1,
						Rtype: "question",
						Rid:   2,
					},
					{
						Sid:   1,
						Slid:  1,
						Rid:   2,
						Rtype: "case",
					},
				}
				for idx, wantReq := range wantReqs {
					s.assertSkillPreRequest(wantReq, reqs[idx])
				}
			},
			req: web.SaveRequestReq{
				Sid:  1,
				Slid: 1,
				Requests: []web.SkillPreRequest{
					{
						Rid:   1,
						Rtype: "case",
					},
					{
						Rid:   2,
						Rtype: "question",
					},
					{
						Rid:   2,
						Rtype: "case",
					},
				},
			},
			wantCode: 200,
		},
		{
			name: "更新",
			before: func(t *testing.T) {
				err := s.db.Create(&dao.Skill{
					Id: 1,
					Labels: sqlx.JsonColumn[[]string]{
						Val:   []string{"old_mysql"},
						Valid: true,
					},
					Name: "redis",
					Desc: "redis_desc",
					Base: dao.Base{
						Ctime: time.Now().UnixMilli(),
						Utime: time.Now().UnixMilli(),
					},
				}).Error
				require.NoError(t, err)
				err = s.db.Create([]*dao.SkillLevel{
					{
						Id:    1,
						Sid:   1,
						Level: "redis_level1",
						Desc:  "redis_desc",
						Base: dao.Base{
							Ctime: time.Now().UnixMilli(),
							Utime: time.Now().UnixMilli(),
						},
					},
				},
				).Error
				require.NoError(t, err)
				err = s.db.Create([]*dao.SkillPreRequest{
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
				reqs, err := s.dao.RequestInfo(ctx, 1)
				require.NoError(t, err)
				wantReqs := []dao.SkillPreRequest{
					{
						Sid:   1,
						Slid:  1,
						Rid:   1,
						Rtype: "case",
					},
					{
						Sid:   1,
						Slid:  1,
						Rtype: "question",
						Rid:   2,
					},
					{
						Sid:   1,
						Slid:  1,
						Rid:   2,
						Rtype: "case",
					},
				}
				for idx, wantReq := range wantReqs {
					s.assertSkillPreRequest(wantReq, reqs[idx])
				}
			},
			req: web.SaveRequestReq{
				Sid:  1,
				Slid: 1,
				Requests: []web.SkillPreRequest{
					{
						Rid:   1,
						Rtype: "case",
					},
					{
						Rid:   2,
						Rtype: "question",
					},
					{
						Rid:   2,
						Rtype: "case",
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
				"/skill/save-request", iox.NewJSONReader(tc.req))
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
			err = s.db.Exec("TRUNCATE  TABLE `skill_pre_request`").Error
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
		Name: "mysql",
		Desc: "mysql_desc",
		Base: dao.Base{
			Ctime: time.Now().UnixMilli(),
			Utime: time.Now().UnixMilli(),
		},
	}).Error
	require.NoError(t, err)
	err = s.db.Create([]*dao.SkillLevel{
		{
			Id:    1,
			Sid:   2,
			Level: "mysql_level1",
			Desc:  "mysql_desc",
			Base: dao.Base{
				Ctime: time.Now().UnixMilli(),
				Utime: time.Now().UnixMilli(),
			},
		},
		{
			Id:    2,
			Sid:   2,
			Level: "mysql_level2",
			Desc:  "mysql_desc",
			Base: dao.Base{
				Ctime: time.Now().UnixMilli(),
				Utime: time.Now().UnixMilli(),
			},
		},
	}).Error
	require.NoError(t, err)
	s.db.Create([]*dao.SkillPreRequest{
		{
			Id:    1,
			Slid:  1,
			Sid:   2,
			Rtype: "case1",
			Rid:   1,
			Base: dao.Base{
				Ctime: time.Now().UnixMilli(),
				Utime: time.Now().UnixMilli(),
			},
		},
		{
			Id:    2,
			Slid:  1,
			Sid:   2,
			Rtype: "q1",
			Rid:   1,
			Base: dao.Base{
				Ctime: time.Now().UnixMilli(),
				Utime: time.Now().UnixMilli(),
			},
		},
		{
			Id:    3,
			Slid:  2,
			Sid:   2,
			Rtype: "q3",
			Rid:   1,
			Base: dao.Base{
				Ctime: time.Now().UnixMilli(),
				Utime: time.Now().UnixMilli(),
			},
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
	s.assertWebSkill(web.Skill{
		ID: 2,
		Labels: []string{
			"mysql",
		},
		Name: "mysql",
		Desc: "mysql_desc",
		Levels: []web.SkillLevel{
			{
				Id:    1,
				Level: "mysql_level1",
				Desc:  "mysql_desc",
				Requests: []web.SkillPreRequest{
					{
						Id:    1,
						Rtype: "case1",
						Rid:   1,
					},
					{
						Id:    2,
						Rtype: "q1",
						Rid:   1,
					},
				},
			},
			{
				Id:    2,
				Level: "mysql_level2",
				Desc:  "mysql_desc",
				Requests: []web.SkillPreRequest{
					{
						Id:    3,
						Rtype: "q3",
						Rid:   1,
					},
				},
			},
		},
	}, recorder.MustScan().Data)
	err = s.db.Exec("TRUNCATE  TABLE `skill`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE `skill_level`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE  TABLE `skill_pre_request`").Error
	require.NoError(s.T(), err)

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
			Name: name,
			Desc: fmt.Sprintf("%s_desc", name),
			Base: dao.Base{
				Ctime: time.Unix(0, 0).UnixMilli(),
				Utime: time.Unix(0, 0).UnixMilli(),
			},
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
						},
						{
							ID:   99,
							Name: "mysql99",
							Desc: "mysql99_desc",
							Labels: []string{
								"mysql99",
							},
							Utime: time.Unix(0, 0).Format(time.DateTime),
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

func (s *HandlerTestSuite) TestPublish() {
	testCases := []struct {
		name   string
		before func(t *testing.T)
		after  func(t *testing.T)
		req    web.SaveReq

		wantCode int
		wantResp test.Result[int64]
	}{
		{
			name: "新增",
			before: func(t *testing.T) {

			},
			after: func(t *testing.T) {
				ctx := context.Background()
				skill, err := s.dao.Info(ctx, 1)
				require.NoError(t, err)
				skillLevels, err := s.dao.SkillLevelInfo(ctx, 1)
				require.NoError(t, err)
				wantSkill := dao.Skill{
					Labels: sqlx.JsonColumn[[]string]{
						Val:   []string{"mysql"},
						Valid: true,
					},
					Name: "mysql",
					Desc: "mysql_desc",
				}
				s.assertSkill(wantSkill, skill)
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
					s.assertSkillLevel(wantLevels[idx], skillLevels[idx])
				}
				pubSkill, err := s.dao.PubInfo(ctx, 1)
				require.NoError(t, err)
				s.assertSkill(wantSkill, dao.Skill(pubSkill))
				pubLevels, err := s.dao.PubLevels(ctx, 1)
				require.NoError(t, err)
				assert.Equal(t, len(wantLevels), len(pubLevels))
				for idx := range pubLevels {
					s.assertSkillLevel(wantLevels[idx], dao.SkillLevel(pubLevels[idx]))
				}

			},
			req: web.SaveReq{
				Skill: web.Skill{
					Labels: []string{"mysql"},
					Name:   "mysql",
					Desc:   "mysql_desc",
					Levels: []web.SkillLevel{
						{
							Level: "basic",
							Desc:  "mysql_basic",
						},
						{
							Level: "intermediate",
							Desc:  "mysql_intermediate",
						},
						{
							Level: "advanced",
							Desc:  "mysql_advanced",
						},
					},
				},
			},
			wantCode: 200,
			wantResp: test.Result[int64]{
				Data: 1,
			},
		},
		{
			name: "从未发布的内容发布",
			before: func(t *testing.T) {
				err := s.db.Create(&dao.Skill{
					Id: 2,
					Labels: sqlx.JsonColumn[[]string]{
						Val:   []string{"old_mysql"},
						Valid: true,
					},
					Name: "old_mysql",
					Desc: "old_mysql_desc",
					Base: dao.Base{
						Ctime: time.Now().UnixMilli(),
						Utime: time.Now().UnixMilli(),
					},
				}).Error
				require.NoError(t, err)
				err = s.db.Create([]*dao.SkillLevel{
					{
						Id:    1,
						Sid:   2,
						Level: "old_mysql_level1",
						Desc:  "old_mysql_desc",
						Base: dao.Base{
							Ctime: time.Now().UnixMilli(),
							Utime: time.Now().UnixMilli(),
						},
					},
					{
						Id:    2,
						Sid:   2,
						Level: "old_mysql_level2",
						Desc:  "old_mysql_desc",
						Base: dao.Base{
							Ctime: time.Now().UnixMilli(),
							Utime: time.Now().UnixMilli(),
						},
					},
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				ctx := context.Background()
				skill, err := s.dao.Info(ctx, 2)
				require.NoError(t, err)
				skillLevels, err := s.dao.SkillLevelInfo(ctx, 2)
				require.NoError(t, err)
				wantSkill := dao.Skill{
					Labels: sqlx.JsonColumn[[]string]{
						Val:   []string{"mysql"},
						Valid: true,
					},
					Name: "mysql",
					Desc: "mysql_desc",
				}
				s.assertSkill(wantSkill, skill)
				wantLevels := []dao.SkillLevel{
					{
						Sid:   2,
						Level: "mysql_level1",
						Desc:  "mysql_desc",
					},
					{
						Sid:   2,
						Level: "mysql_level2",
						Desc:  "mysql_desc",
					},
					{
						Sid:   2,
						Level: "mysql_level3",
						Desc:  "mysql_desc",
					},
				}
				assert.Equal(t, len(wantLevels), len(skillLevels))
				for idx := range skillLevels {
					s.assertSkillLevel(wantLevels[idx], skillLevels[idx])
				}
				pubSkill, err := s.dao.PubInfo(ctx, 2)
				require.NoError(t, err)
				s.assertSkill(wantSkill, dao.Skill(pubSkill))
				pubLevels, err := s.dao.PubLevels(ctx, 2)
				require.NoError(t, err)
				assert.Equal(t, len(wantLevels), len(pubLevels))
				for idx := range pubLevels {
					s.assertSkillLevel(wantLevels[idx], dao.SkillLevel(pubLevels[idx]))
				}

			},
			req: web.SaveReq{
				Skill: web.Skill{
					ID:     2,
					Labels: []string{"mysql"},
					Name:   "mysql",
					Desc:   "mysql_desc",
					Levels: []web.SkillLevel{
						{
							Id:    1,
							Level: "mysql_level1",
							Desc:  "mysql_desc",
						},
						{
							Id:    2,
							Level: "mysql_level2",
							Desc:  "mysql_desc",
						},
						{
							Level: "mysql_level3",
							Desc:  "mysql_desc",
						},
					},
				},
			},
			wantCode: 200,
			wantResp: test.Result[int64]{
				Data: 2,
			},
		},
		{
			name: "已发布的内容修改",
			before: func(t *testing.T) {
				err := s.db.Create(&dao.Skill{
					Id: 3,
					Labels: sqlx.JsonColumn[[]string]{
						Val:   []string{"old_mysql"},
						Valid: true,
					},
					Name: "old_mysql",
					Desc: "old_mysql_desc",
					Base: dao.Base{
						Ctime: time.Now().UnixMilli(),
						Utime: time.Now().UnixMilli(),
					},
				}).Error
				require.NoError(t, err)
				err = s.db.Create([]*dao.SkillLevel{
					{
						Id:    1,
						Sid:   3,
						Level: "old_mysql_level1",
						Desc:  "old_mysql_desc",
						Base: dao.Base{
							Ctime: time.Now().UnixMilli(),
							Utime: time.Now().UnixMilli(),
						},
					},
					{
						Id:    2,
						Sid:   3,
						Level: "old_mysql_level2",
						Desc:  "old_mysql_desc",
						Base: dao.Base{
							Ctime: time.Now().UnixMilli(),
							Utime: time.Now().UnixMilli(),
						},
					},
				}).Error
				require.NoError(t, err)
				err = s.db.Create([]*dao.PubSkill{
					{
						Id: 3,
						Labels: sqlx.JsonColumn[[]string]{
							Val:   []string{"old_mysql"},
							Valid: true,
						},
						Name: "old_mysql",
						Desc: "old_mysql_desc",
						Base: dao.Base{
							Ctime: time.Now().UnixMilli(),
							Utime: time.Now().UnixMilli(),
						},
					},
				}).Error
				require.NoError(t, err)
				err = s.db.Create([]*dao.PubSkillLevel{
					{
						Id:    1,
						Sid:   3,
						Level: "old_mysql_level1",
						Desc:  "old_mysql_desc",
						Base: dao.Base{
							Ctime: time.Now().UnixMilli(),
							Utime: time.Now().UnixMilli(),
						},
					},
					{
						Id:    2,
						Sid:   3,
						Level: "old_mysql_level2",
						Desc:  "old_mysql_desc",
						Base: dao.Base{
							Ctime: time.Now().UnixMilli(),
							Utime: time.Now().UnixMilli(),
						},
					},
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				ctx := context.Background()
				skill, err := s.dao.Info(ctx, 3)
				require.NoError(t, err)
				skillLevels, err := s.dao.SkillLevelInfo(ctx, 3)
				require.NoError(t, err)
				wantSkill := dao.Skill{
					Labels: sqlx.JsonColumn[[]string]{
						Val:   []string{"mysql"},
						Valid: true,
					},
					Name: "mysql",
					Desc: "mysql_desc",
				}
				s.assertSkill(wantSkill, skill)
				wantLevels := []dao.SkillLevel{
					{
						Sid:   3,
						Level: "mysql_level1",
						Desc:  "mysql_desc",
					},
					{
						Sid:   3,
						Level: "mysql_level2",
						Desc:  "mysql_desc",
					},
					{
						Sid:   3,
						Level: "mysql_level3",
						Desc:  "mysql_desc",
					},
				}
				assert.Equal(t, len(wantLevels), len(skillLevels))
				for idx := range skillLevels {
					s.assertSkillLevel(wantLevels[idx], skillLevels[idx])
				}
				pubSkill, err := s.dao.PubInfo(ctx, 3)
				require.NoError(t, err)
				s.assertSkill(wantSkill, dao.Skill(pubSkill))
				pubLevels, err := s.dao.PubLevels(ctx, 3)
				require.NoError(t, err)
				assert.Equal(t, len(wantLevels), len(pubLevels))
				for idx := range pubLevels {
					s.assertSkillLevel(wantLevels[idx], dao.SkillLevel(pubLevels[idx]))
				}
			},
			req: web.SaveReq{
				Skill: web.Skill{
					ID:     3,
					Labels: []string{"mysql"},
					Name:   "mysql",
					Desc:   "mysql_desc",
					Levels: []web.SkillLevel{
						{
							Id:    1,
							Level: "mysql_level1",
							Desc:  "mysql_desc",
						},
						{
							Id:    2,
							Level: "mysql_level2",
							Desc:  "mysql_desc",
						},
						{
							Level: "mysql_level3",
							Desc:  "mysql_desc",
						},
					},
				},
			},
			wantCode: 200,
			wantResp: test.Result[int64]{
				Data: 3,
			},
		},
	}
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/skill/publish", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[int64]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
			tc.after(t)
			err = s.db.Exec("TRUNCATE table `skill`").Error
			require.NoError(t, err)
			err = s.db.Exec("TRUNCATE table `pub_skill`").Error
			require.NoError(t, err)
			err = s.db.Exec("TRUNCATE table `skill_level`").Error
			require.NoError(t, err)
			err = s.db.Exec("TRUNCATE table `pub_skill_level`").Error
			require.NoError(t, err)
		})
	}
}

func (s *HandlerTestSuite) TestPublishReq() {
	testCases := []struct {
		name     string
		before   func(t *testing.T)
		after    func(t *testing.T)
		req      web.SaveRequestReq
		wantCode int
	}{
		{
			name:   "新建",
			before: func(t *testing.T) {},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				reqs, err := s.dao.RequestInfo(ctx, 1)
				require.NoError(t, err)
				pubReqs, err := s.dao.PubRequestInfo(ctx, 1)
				require.NoError(t, err)
				wantReqs := []dao.SkillPreRequest{
					{
						Sid:   1,
						Slid:  1,
						Rid:   1,
						Rtype: "case",
					},
					{
						Sid:   1,
						Slid:  1,
						Rtype: "question",
						Rid:   2,
					},
					{
						Sid:   1,
						Slid:  1,
						Rid:   2,
						Rtype: "case",
					},
				}
				for idx, wantReq := range wantReqs {
					s.assertSkillPreRequest(wantReq, reqs[idx])
					s.assertSkillPreRequest(wantReq, dao.SkillPreRequest(pubReqs[idx]))
				}

			},
			req: web.SaveRequestReq{
				Sid:  1,
				Slid: 1,
				Requests: []web.SkillPreRequest{
					{
						Rid:   1,
						Rtype: "case",
					},
					{
						Rid:   2,
						Rtype: "question",
					},
					{
						Rid:   2,
						Rtype: "case",
					},
				},
			},
			wantCode: 200,
		},
		{
			name: "更新",
			before: func(t *testing.T) {
				err := s.db.Create([]*dao.PubSKillPreRequest{
					{
						Sid:   2,
						Slid:  1,
						Rtype: "question",
						Rid:   2,
					},
					{
						Sid:   2,
						Slid:  1,
						Rid:   1,
						Rtype: "case",
					},
				}).Error
				require.NoError(t, err)
				err = s.db.Create([]*dao.SkillPreRequest{
					{
						Sid:   2,
						Slid:  1,
						Rtype: "question",
						Rid:   2,
					},
					{
						Sid:   2,
						Slid:  1,
						Rid:   1,
						Rtype: "case",
					},
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				reqs, err := s.dao.RequestInfo(ctx, 2)
				require.NoError(t, err)
				pubReqs, err := s.dao.PubRequestInfo(ctx, 2)
				require.NoError(t, err)
				wantReqs := []dao.SkillPreRequest{
					{
						Sid:   2,
						Slid:  1,
						Rid:   1,
						Rtype: "case1",
					},
					{
						Sid:   2,
						Slid:  1,
						Rtype: "question1",
						Rid:   2,
					},
					{
						Sid:   2,
						Slid:  1,
						Rid:   2,
						Rtype: "case2",
					},
				}
				for idx, wantReq := range wantReqs {
					s.assertSkillPreRequest(wantReq, reqs[idx])
					s.assertSkillPreRequest(wantReq, dao.SkillPreRequest(pubReqs[idx]))
				}

			},
			req: web.SaveRequestReq{
				Sid:  2,
				Slid: 1,
				Requests: []web.SkillPreRequest{
					{
						Rid:   1,
						Rtype: "case1",
					},
					{
						Rid:   2,
						Rtype: "question1",
					},
					{
						Rid:   2,
						Rtype: "case2",
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
				"/skill/publish-request", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[any]()
			s.server.ServeHTTP(recorder, req)
			tc.after(t)
			// 清理数据
			err = s.db.Exec("TRUNCATE  TABLE `skill_pre_request`").Error
			require.NoError(s.T(), err)
			err = s.db.Exec("TRUNCATE TABLE `pub_skill_pre_request`").Error
			require.NoError(s.T(), err)
		})
	}
}

func (s *HandlerTestSuite) TestPublishDetail() {
	t := s.T()
	err := s.db.Create(&dao.PubSkill{
		Id: 2,
		Labels: sqlx.JsonColumn[[]string]{
			Val:   []string{"mysql"},
			Valid: true,
		},
		Name: "mysql",
		Desc: "mysql_desc",
		Base: dao.Base{
			Ctime: time.Now().UnixMilli(),
			Utime: time.Now().UnixMilli(),
		},
	}).Error
	require.NoError(t, err)
	err = s.db.Create([]*dao.PubSkillLevel{
		{
			Id:    1,
			Sid:   2,
			Level: "mysql_level1",
			Desc:  "mysql_desc",
			Base: dao.Base{
				Ctime: time.Now().UnixMilli(),
				Utime: time.Now().UnixMilli(),
			},
		},
		{
			Id:    2,
			Sid:   2,
			Level: "mysql_level2",
			Desc:  "mysql_desc",
			Base: dao.Base{
				Ctime: time.Now().UnixMilli(),
				Utime: time.Now().UnixMilli(),
			},
		},
	}).Error
	require.NoError(t, err)
	s.db.Create([]*dao.PubSKillPreRequest{
		{
			Id:    1,
			Slid:  1,
			Sid:   2,
			Rtype: "case1",
			Rid:   1,
			Base: dao.Base{
				Ctime: time.Now().UnixMilli(),
				Utime: time.Now().UnixMilli(),
			},
		},
		{
			Id:    2,
			Slid:  1,
			Sid:   2,
			Rtype: "q1",
			Rid:   1,
			Base: dao.Base{
				Ctime: time.Now().UnixMilli(),
				Utime: time.Now().UnixMilli(),
			},
		},
		{
			Id:    3,
			Slid:  2,
			Sid:   2,
			Rtype: "q3",
			Rid:   1,
			Base: dao.Base{
				Ctime: time.Now().UnixMilli(),
				Utime: time.Now().UnixMilli(),
			},
		},
	})
	sid := web.Sid{
		Sid: 2,
	}
	req, err := http.NewRequest(http.MethodPost,
		"/skill/pub/detail", iox.NewJSONReader(sid))
	req.Header.Set("content-type", "application/json")
	require.NoError(t, err)
	recorder := test.NewJSONResponseRecorder[web.Skill]()
	s.server.ServeHTTP(recorder, req)
	require.Equal(t, 200, recorder.Code)
	s.assertWebSkill(web.Skill{
		ID: 2,
		Labels: []string{
			"mysql",
		},
		Name: "mysql",
		Desc: "mysql_desc",
		Levels: []web.SkillLevel{
			{
				Id:    1,
				Level: "mysql_level1",
				Desc:  "mysql_desc",
				Requests: []web.SkillPreRequest{
					{
						Id:    1,
						Rtype: "case1",
						Rid:   1,
					},
					{
						Id:    2,
						Rtype: "q1",
						Rid:   1,
					},
				},
			},
			{
				Id:    2,
				Level: "mysql_level2",
				Desc:  "mysql_desc",
				Requests: []web.SkillPreRequest{
					{
						Id:    3,
						Rtype: "q3",
						Rid:   1,
					},
				},
			},
		},
	}, recorder.MustScan().Data)
	err = s.db.Exec("TRUNCATE  TABLE `pub_skill`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE `pub_skill_level`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE  TABLE `pub_skill_pre_request`").Error
	require.NoError(s.T(), err)

}

func (s *HandlerTestSuite) TestPubList() {
	skills := make([]*dao.PubSkill, 0, 100)
	for i := 1; i <= 100; i++ {
		name := fmt.Sprintf("mysql%d", i)
		skills = append(skills, &dao.PubSkill{
			Id: int64(i),
			Labels: sqlx.JsonColumn[[]string]{
				Val:   []string{name},
				Valid: true,
			},
			Name: name,
			Desc: fmt.Sprintf("%s_desc", name),
			Base: dao.Base{
				Ctime: time.Unix(0, 0).UnixMilli(),
				Utime: time.Unix(0, 0).UnixMilli(),
			},
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
						},
						{
							ID:   99,
							Name: "mysql99",
							Desc: "mysql99_desc",
							Labels: []string{
								"mysql99",
							},
							Utime: time.Unix(0, 0).Format(time.DateTime),
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
						},
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost,
				"/skill/pub/list", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.SkillList]()
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

func (s *HandlerTestSuite) assertSkillLevel(wantSKillLevel dao.SkillLevel, actualSkillLevel dao.SkillLevel) {
	t := s.T()
	require.True(t, actualSkillLevel.Id > 0)
	require.True(t, actualSkillLevel.Utime > 0)
	require.True(t, actualSkillLevel.Ctime > 0)
	actualSkillLevel.Id = 0
	actualSkillLevel.Utime = 0
	actualSkillLevel.Ctime = 0
	assert.Equal(t, wantSKillLevel, actualSkillLevel)
}

func (s *HandlerTestSuite) assertSkillPreRequest(wantReq dao.SkillPreRequest, actualReq dao.SkillPreRequest) {
	t := s.T()
	require.True(t, actualReq.Id > 0)
	require.True(t, actualReq.Ctime > 0)
	require.True(t, actualReq.Utime > 0)
	actualReq.Id = 0
	actualReq.Utime = 0
	actualReq.Ctime = 0
	assert.Equal(t, wantReq, actualReq)
}

func (s *HandlerTestSuite) assertWebSkill(wantSkill web.Skill, actualSkill web.Skill) {
	t := s.T()
	require.True(t, actualSkill.Utime != "")
	actualSkill.Utime = ""
	for i := range actualSkill.Levels {
		require.True(t, actualSkill.Levels[i].Utime != "")
		actualSkill.Levels[i].Utime = ""
		for j := range actualSkill.Levels[i].Requests {
			require.True(t, actualSkill.Levels[i].Requests[j].Utime != "")
			actualSkill.Levels[i].Requests[j].Utime = ""
		}
	}
	assert.Equal(t, wantSkill, actualSkill)
}

func TestHandler(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}
