//go:build e2e

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
	"github.com/ecodeclub/webook/internal/interactive"
	intrmocks "github.com/ecodeclub/webook/internal/interactive/mocks"
	"github.com/ecodeclub/webook/internal/permission"
	"github.com/ecodeclub/webook/internal/question/internal/domain"
	"github.com/ecodeclub/webook/internal/question/internal/integration/startup"
	"github.com/ecodeclub/webook/internal/question/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/test"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/econf"
	"github.com/gotomicro/ego/server/egin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type KnowledgeBaseTestSuite struct {
	BaseTestSuite
}

func (k *KnowledgeBaseTestSuite) SetupSuite() {
	k.db = testioc.InitDB()
}

func getWantQuestion(id int64) domain.Question {
	que := domain.Question{
		Id:      id,
		Uid:     uid,
		Biz:     domain.QuestionBiz,
		BizId:   id,
		Title:   fmt.Sprintf("标题%d", id),
		Content: fmt.Sprintf("内容%d", id),
		Status:  domain.PublishedStatus,
		Answer: domain.Answer{
			Analysis:     getAnswerElement(id),
			Basic:        getAnswerElement(id),
			Intermediate: getAnswerElement(id),
			Advanced:     getAnswerElement(id),
		},
		Utime: time.UnixMilli(123),
	}
	return que
}

func getAnswerElement(idx int64) domain.AnswerElement {
	return domain.AnswerElement{
		Content:   fmt.Sprintf("这是解析 %d", idx),
		Keywords:  fmt.Sprintf("关键字 %d", idx),
		Shorthand: fmt.Sprintf("快速记忆法 %d", idx),
		Highlight: fmt.Sprintf("亮点 %d", idx),
		Guidance:  fmt.Sprintf("引导点 %d", idx),
	}
}

func (k *KnowledgeBaseTestSuite) TestKnowledgeBaseSync() {
	ctrl := gomock.NewController(k.T())
	svc := aimocks.NewMockRepositoryBaseSvc(ctrl)
	svc.EXPECT().UploadFile(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, file ai.KnowledgeBaseFile) error {
		wantQue := getWantQuestion(file.BizID)
		var actualQue domain.Question
		err := json.Unmarshal(file.Data, &actualQue)
		if err != nil {
			return err
		}
		actualQue.Answer.Advanced.Id = 0
		actualQue.Answer.Analysis.Id = 0
		actualQue.Answer.Basic.Id = 0
		actualQue.Answer.Intermediate.Id = 0
		wantQue.Utime = actualQue.Utime
		assert.Equal(k.T(), wantQue, actualQue)
		return nil
	}).AnyTimes()
	// 初始化数据
	err := dao.InitTables(k.db)
	require.NoError(k.T(), err)
	quedao := k.buildQuestion(3)
	quedao.Biz = domain.QuestionBiz
	quedao.Status = domain.PublishedStatus.ToUint8()
	analysis := k.buildDAOAnswerEle(3, 3, dao.AnswerElementTypeAnalysis)
	basic := k.buildDAOAnswerEle(3, 3, dao.AnswerElementTypeBasic)
	inter := k.buildDAOAnswerEle(3, 3, dao.AnswerElementTypeIntermedia)
	advance := k.buildDAOAnswerEle(3, 3, dao.AnswerElementTypeAdvanced)
	k.db.WithContext(context.Background()).Model(&dao.Question{}).Create(&quedao)
	err = k.db.WithContext(context.Background()).Model(&dao.AnswerElement{}).
		Create([]dao.AnswerElement{
			analysis,
			basic,
			inter,
			advance,
		}).Error
	require.NoError(k.T(), err)

	intrSvc := intrmocks.NewMockService(ctrl)
	intrModule := &interactive.Module{
		Svc: intrSvc,
	}
	module, err := startup.InitModule(nil, nil, intrModule, &permission.Module{}, &ai.Module{
		KnowledgeBaseSvc: svc,
	},
		session.DefaultProvider(),
		&member.Module{})
	require.NoError(k.T(), err)
	econf.Set("server", map[string]any{"contextTimeout": "1s"})
	server := egin.Load("server").Build()
	server.Use(func(ctx *gin.Context) {
		ctx.Set("_session", session.NewMemorySession(session.Claims{
			Uid: uid,
			Data: map[string]string{
				"creator":   "true",
				"memberDDL": strconv.FormatInt(time.Now().Add(time.Hour).UnixMilli(), 10),
			},
		}))
	})
	module.KnowledgeBaseHdl.PrivateRoutes(server.Engine)

	req, err := http.NewRequest(http.MethodGet,
		"/question/knowledgeBase/syncAll", iox.NewJSONReader(nil))
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
