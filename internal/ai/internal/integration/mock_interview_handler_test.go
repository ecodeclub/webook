package integration

import (
	"os"
	"strings"
	"testing"

	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/ai/internal/integration/startup"
	"github.com/ecodeclub/webook/internal/ai/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/ai/internal/web"
	"github.com/ecodeclub/webook/internal/credit"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/econf"
	"github.com/gotomicro/ego/core/elog"
	"github.com/gotomicro/ego/server/egin"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap/zapcore"
	"gorm.io/gorm"
)

func TestMockInterview(t *testing.T) {
	suite.Run(t, new(MockInterviewTestSuite))
}

type MockInterviewTestSuite struct {
	suite.Suite
	db               *gorm.DB
	mockInterviewHdl *web.MockInterviewHandler
	server           *egin.Component
}

func (s *MockInterviewTestSuite) SetupSuite() {
	db := testioc.InitDB()
	s.db = db
	err := dao.InitTables(db)
	s.NoError(err)
	// 先插入 BizConfig
	mou, err := startup.InitModule(s.db, nil, nil, nil, &credit.Module{}, nil)
	s.NoError(err)
	s.mockInterviewHdl = mou.MockInterviewHdl

	econf.Set("server", map[string]any{"contextTimeout": "10m"})
	server := egin.Load("server").Build()

	// 添加 CORS 配置（与 gin.go 中的配置保持一致）
	server.Use(cors.New(cors.Config{
		ExposeHeaders:    []string{"X-Refresh-Token", "X-Access-Token"},
		AllowCredentials: true,
		AllowHeaders: []string{"X-Timestamp",
			"X-APP",
			"Authorization", "Content-Type"},
		AllowOriginFunc: func(origin string) bool {
			if strings.HasPrefix(origin, "http://localhost") {
				return true
			}
			// 允许本地开发环境
			return strings.Contains(origin, "localhost:63342") ||
				strings.Contains(origin, "127.0.0.1")
		},
	}))

	// 添加 session 模拟
	server.Use(func(ctx *gin.Context) {
		ctx.Set("_session", session.NewMemorySession(session.Claims{
			Uid: 123,
		}))
	})

	s.mockInterviewHdl.PrivateRoutes(server.Engine)
	s.server = server
}

func (s *MockInterviewTestSuite) TestE2E() {
	elog.DefaultLogger.SetLevel(zapcore.DebugLevel)
	t := s.T()

	cosSecretID := os.Getenv("COS_SECRET_ID")
	if cosSecretID == "" {
		t.Fatal("未设置 COS_SECRET_ID 环境变量")
	}
	econf.Set("cos.secretID", cosSecretID)

	cosSecretKey := os.Getenv("COS_SECRET_KEY")
	if cosSecretKey == "" {
		t.Fatal("未设置 COS_SECRET_KEY 环境变量")
	}
	econf.Set("cos.secretKey", cosSecretKey)

	cosBucket := os.Getenv("COS_BUCKET_NAME")
	if cosBucket == "" {
		t.Fatal("未设置 COS_BUCKET_NAME 环境变量（格式：bucketname-appid）")
	}
	econf.Set("cos.bucket", cosBucket)

	cosRegion := os.Getenv("COS_REGION")
	if cosRegion == "" {
		t.Fatal("未设置 COS_REGION 环境变量（如：ap-guangzhou）")
	}
	econf.Set("cos.region", cosRegion)

	if err := s.server.Run("localhost:8080"); err != nil {
		t.Fatal(err)
	}
}
