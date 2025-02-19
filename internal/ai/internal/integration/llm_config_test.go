package integration

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/ecodeclub/ekit/iox"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/ai/internal/integration/startup"
	"github.com/ecodeclub/webook/internal/ai/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/ai/internal/web"
	"github.com/ecodeclub/webook/internal/credit"
	"github.com/ecodeclub/webook/internal/test"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/econf"
	"github.com/gotomicro/ego/server/egin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type ConfigSuite struct {
	suite.Suite
	db           *gorm.DB
	adminHandler *web.AdminHandler
	server       *egin.Component
}

func (s *ConfigSuite) SetupSuite() {
	db := testioc.InitDB()
	s.db = db
	err := dao.InitTables(db)
	s.NoError(err)
	// 先插入 BizConfig
	mou, err := startup.InitModule(s.db, nil, nil, nil, &credit.Module{}, nil)
	require.NoError(s.T(), err)
	s.adminHandler = mou.AdminHandler
	econf.Set("server", map[string]any{"contextTimeout": "1s"})
	server := egin.Load("server").Build()
	server.Use(func(ctx *gin.Context) {
		ctx.Set("_session", session.NewMemorySession(session.Claims{
			Uid: 123,
		}))
	})
	s.adminHandler.RegisterRoutes(server.Engine)
	s.server = server
}

func (s *ConfigSuite) TestConfig_Save() {
	testCases := []struct {
		name     string
		config   web.ConfigRequest
		before   func(t *testing.T)
		after    func(t *testing.T, id int64)
		wantCode int
		id       int64
	}{
		{
			name: "新增",
			config: web.ConfigRequest{
				Config: web.Config{
					Biz:            "test",
					MaxInput:       10,
					Model:          "testModel",
					Price:          100,
					Temperature:    0.5,
					TopP:           0.5,
					SystemPrompt:   "testPrompt",
					PromptTemplate: "testTemplate",
					KnowledgeId:    "testKnowledgeId",
				},
			},
			before: func(t *testing.T) {

			},
			wantCode: 200,
			id:       1,
			after: func(t *testing.T, id int64) {
				var conf dao.BizConfig
				err := s.db.WithContext(context.Background()).
					Where("id = ?", id).First(&conf).Error
				require.NoError(t, err)
				s.assertBizConfig(dao.BizConfig{
					Id:             1,
					Biz:            "test",
					MaxInput:       10,
					Model:          "testModel",
					Price:          100,
					Temperature:    0.5,
					TopP:           0.5,
					SystemPrompt:   "testPrompt",
					PromptTemplate: "testTemplate",
					KnowledgeId:    "testKnowledgeId",
				}, conf)
			},
		},
		{
			name: "更新",
			config: web.ConfigRequest{
				Config: web.Config{
					Id:             2,
					Biz:            "2_test",
					MaxInput:       102,
					Model:          "2_testModel",
					Price:          102,
					Temperature:    2.5,
					TopP:           2.5,
					SystemPrompt:   "testPrompt2",
					PromptTemplate: "testTemplate2",
					KnowledgeId:    "testKnowledgeId2",
				},
			},
			before: func(t *testing.T) {
				err := s.db.WithContext(context.Background()).
					Table("ai_biz_configs").
					Create(dao.BizConfig{
						Id:             2,
						Biz:            "test_2",
						MaxInput:       100,
						Model:          "testModel",
						Price:          100,
						Temperature:    0.5,
						TopP:           0.5,
						SystemPrompt:   "testPrompt",
						PromptTemplate: "testTemplate",
						KnowledgeId:    "testKnowledgeId",
						Ctime:          11,
						Utime:          22,
					}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T, id int64) {
				var conf dao.BizConfig
				err := s.db.WithContext(context.Background()).
					Where("id = ?", id).
					Model(&dao.BizConfig{}).
					First(&conf).Error
				require.NoError(t, err)
				s.assertBizConfig(dao.BizConfig{
					Id:             2,
					Biz:            "2_test",
					MaxInput:       102,
					Model:          "2_testModel",
					Price:          102,
					Temperature:    2.5,
					TopP:           2.5,
					SystemPrompt:   "testPrompt2",
					PromptTemplate: "testTemplate2",
					KnowledgeId:    "testKnowledgeId2",
				}, conf)
			},
			wantCode: 200,
			id:       2,
		},
	}
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/ai/config/save", iox.NewJSONReader(tc.config))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[int64]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			id := recorder.MustScan().Data
			assert.Equal(t, tc.id, id)
			tc.after(t, id)
			err = s.db.Exec("TRUNCATE TABLE `ai_biz_configs`").Error
			require.NoError(s.T(), err)
		})
	}
}

func (s *ConfigSuite) TestConfig_List() {
	configs := make([]dao.BizConfig, 0, 32)
	for i := 1; i < 10; i++ {
		cfg := dao.BizConfig{
			Id:             int64(i),
			Biz:            fmt.Sprintf("biz_%d", i),
			MaxInput:       100,
			Model:          fmt.Sprintf("test_model_%d", i),
			Price:          1000,
			Temperature:    37.5,
			TopP:           0.8,
			SystemPrompt:   "test_prompt",
			PromptTemplate: "test_template",
			KnowledgeId:    "test_knowledge",
			Utime:          int64(i),
		}
		configs = append(configs, cfg)
	}
	err := s.db.WithContext(context.Background()).Create(&configs).Error
	require.NoError(s.T(), err)
	req, err := http.NewRequest(http.MethodGet,
		"/ai/config/list", iox.NewJSONReader(nil))
	req.Header.Set("content-type", "application/json")
	require.NoError(s.T(), err)
	recorder := test.NewJSONResponseRecorder[[]web.Config]()
	s.server.ServeHTTP(recorder, req)
	require.Equal(s.T(), 200, recorder.Code)
	confs := recorder.MustScan().Data
	assert.Equal(s.T(), getWantConfigs(), confs)
	err = s.db.Exec("TRUNCATE TABLE `ai_biz_configs`").Error
	require.NoError(s.T(), err)
}

func (s *ConfigSuite) Test_Detail() {
	testcases := []struct {
		name     string
		req      web.ConfigInfoReq
		before   func(t *testing.T)
		wantCode int
		wantData web.Config
	}{
		{
			name:     "获取配置",
			wantCode: 200,
			req: web.ConfigInfoReq{
				Id: 3,
			},
			before: func(t *testing.T) {
				err := s.db.WithContext(context.Background()).
					Table("ai_biz_configs").
					Create(dao.BizConfig{
						Id:             3,
						Biz:            "test_3",
						MaxInput:       100,
						Model:          "testModel",
						Price:          100,
						Temperature:    0.5,
						TopP:           0.5,
						SystemPrompt:   "testPrompt",
						PromptTemplate: "testTemplate",
						KnowledgeId:    "testKnowledgeId",
						Ctime:          11,
						Utime:          22,
					}).Error
				require.NoError(t, err)
			},
			wantData: web.Config{
				Id:             3,
				Biz:            "test_3",
				MaxInput:       100,
				Model:          "testModel",
				Price:          100,
				Temperature:    0.5,
				TopP:           0.5,
				SystemPrompt:   "testPrompt",
				PromptTemplate: "testTemplate",
				KnowledgeId:    "testKnowledgeId",
				Utime:          22,
			},
		},
	}
	for _, tc := range testcases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/ai/config/detail", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.Config]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(s.T(), 200, recorder.Code)
			conf := recorder.MustScan().Data
			assert.Equal(t, tc.wantData, conf)
			err = s.db.Exec("TRUNCATE TABLE `ai_biz_configs`").Error
			require.NoError(s.T(), err)
		})
	}
}

func getWantConfigs() []web.Config {
	configs := make([]web.Config, 0, 32)
	for i := 9; i >= 1; i-- {
		cfg := web.Config{
			Id:             int64(i),
			Biz:            fmt.Sprintf("biz_%d", i),
			MaxInput:       100,
			Model:          fmt.Sprintf("test_model_%d", i),
			Price:          1000,
			Temperature:    37.5,
			TopP:           0.8,
			SystemPrompt:   "test_prompt",
			PromptTemplate: "test_template",
			KnowledgeId:    "test_knowledge",
			Utime:          int64(i),
		}
		configs = append(configs, cfg)
	}
	return configs
}

func (s *ConfigSuite) assertBizConfig(wantConfig dao.BizConfig, actualConfig dao.BizConfig) {
	assert.True(s.T(), actualConfig.Ctime > 0)
	assert.True(s.T(), actualConfig.Utime > 0)
	actualConfig.Ctime = 0
	actualConfig.Utime = 0
	assert.Equal(s.T(), wantConfig, actualConfig)
}

func TestConfigSuite(t *testing.T) {
	suite.Run(t, new(ConfigSuite))
}
