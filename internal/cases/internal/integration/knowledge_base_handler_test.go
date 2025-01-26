package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/ecodeclub/webook/internal/member"

	"github.com/ecodeclub/ekit/iox"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/ai"
	aimocks "github.com/ecodeclub/webook/internal/ai/mocks"
	"github.com/ecodeclub/webook/internal/cases/internal/domain"
	eveMocks "github.com/ecodeclub/webook/internal/cases/internal/event/mocks"
	"github.com/ecodeclub/webook/internal/cases/internal/integration/startup"
	"github.com/ecodeclub/webook/internal/cases/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/cases/internal/service"
	"github.com/ecodeclub/webook/internal/interactive"
	intrmocks "github.com/ecodeclub/webook/internal/interactive/mocks"
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
)

type KnowledgeBaseTestSuite struct {
	suite.Suite
	db      *egorm.Component
	caseSvc service.Service
}

func (k *KnowledgeBaseTestSuite) SetupSuite() {
	k.db = testioc.InitDB()
}

func (k *KnowledgeBaseTestSuite) getWantCase(id int64) domain.Case {
	ca := domain.Case{
		Id:           id,
		Uid:          123,
		Labels:       []string{"label"},
		Introduction: fmt.Sprintf("intro %d", id),
		Title:        fmt.Sprintf("标题%d", id),
		Content:      fmt.Sprintf("内容%d", id),
		GiteeRepo:    "gitee",
		GithubRepo:   "github",
		Shorthand:    "速记",
		Keywords:     fmt.Sprintf("关键字 %d", id),
		Highlight:    fmt.Sprintf("亮点 %d", id),
		Guidance:     fmt.Sprintf("引导点 %d", id),
		Biz:          domain.BizCase,
		BizId:        id,
		Status:       domain.PublishedStatus,
	}
	return ca
}

func (k *KnowledgeBaseTestSuite) TestKnowledgeBaseSync() {
	ctrl := gomock.NewController(k.T())
	svc := aimocks.NewMockRepositoryBaseSvc(ctrl)
	producer := eveMocks.NewMockSyncEventProducer(ctrl)
	knowledgeProducer := eveMocks.NewMockKnowledgeBaseEventProducer(ctrl)
	producer.EXPECT().Produce(gomock.Any(), gomock.Any()).AnyTimes()
	knowledgeProducer.EXPECT().Produce(gomock.Any(), gomock.Any()).AnyTimes()
	svc.EXPECT().UploadFile(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, file ai.KnowledgeBaseFile) error {
		assert.Equal(k.T(), fmt.Sprintf("case_%d", file.BizID), file.Name)
		assert.Equal(k.T(), domain.BizCase, file.Biz)
		wantCase := k.getWantCase(file.BizID)
		var actualCa domain.Case
		err := json.Unmarshal(file.Data, &actualCa)
		if err != nil {
			return err
		}
		assert.Equal(k.T(), file.BizID, wantCase.Id)
		actualCa.Ctime = wantCase.Ctime
		actualCa.Utime = wantCase.Utime
		assert.Equal(k.T(), wantCase, actualCa)
		return nil
	}).AnyTimes()
	// 初始化数据
	err := dao.InitTables(k.db)
	require.NoError(k.T(), err)

	require.NoError(k.T(), err)

	intrSvc := intrmocks.NewMockService(ctrl)
	intrModule := &interactive.Module{
		Svc: intrSvc,
	}
	module, err := startup.InitModule(producer, knowledgeProducer, &ai.Module{
		KnowledgeBaseSvc: svc,
	}, &member.Module{},
		session.DefaultProvider(),
		intrModule)
	require.NoError(k.T(), err)
	k.caseSvc = module.Svc
	wantCa := k.getWantCase(1)
	_, err = k.caseSvc.Publish(context.Background(), wantCa)
	require.NoError(k.T(), err)

	econf.Set("server", map[string]any{"contextTimeout": "1s"})
	server := egin.Load("server").Build()
	server.Use(func(ctx *gin.Context) {
		ctx.Set("_session", session.NewMemorySession(session.Claims{
			Uid: 123,
			Data: map[string]string{
				"creator":   "true",
				"memberDDL": strconv.FormatInt(time.Now().Add(time.Hour).UnixMilli(), 10),
			},
		}))
	})
	module.KnowledgeBaseHandler.PrivateRoutes(server.Engine)

	req, err := http.NewRequest(http.MethodGet,
		"/case/knowledgeBase/syncAll", iox.NewJSONReader(nil))
	req.Header.Set("content-type", "application/json")
	require.NoError(k.T(), err)
	recorder := test.NewJSONResponseRecorder[int64]()
	server.ServeHTTP(recorder, req)
	// 等待同步完成
	time.Sleep(3 * time.Second)
}

func TestKnowledgeBaseTestSuite(t *testing.T) {
	suite.Run(t, new(KnowledgeBaseTestSuite))
}
