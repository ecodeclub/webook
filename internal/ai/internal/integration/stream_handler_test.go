package integration

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/ecodeclub/webook/internal/ai/internal/service/llm/handler/platform/ali_deepseek"

	"github.com/ecodeclub/webook/internal/ai/internal/domain"
	"github.com/ecodeclub/webook/internal/ai/internal/repository"
	"github.com/ecodeclub/webook/internal/ai/internal/repository/dao"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/stretchr/testify/require"
)

func TestHandler_StreamHandle(t *testing.T) {
	//t.Skip()
	handler := initHandler(t)
	msgChan, err := handler.StreamHandle(context.Background(), domain.LLMRequest{
		Biz:   "case",
		Uid:   23,
		Tid:   "tid1",
		Input: []string{"上海"},
	})
	require.NoError(t, err)
	// 修改后：
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Second)
	defer cancel()
	for {
		select {
		case event, ok := <-msgChan:
			if !ok {
				// 通道关闭时退出
				log.Println("通道关闭")
				return
			}
			if event.Done {
				log.Println("对话停止")
			}
			log.Printf("\ncontent: %s\n reasoning_content: %s", event.Content, event.ReasoningContent)
		case <-ctx.Done():
			log.Println("超时关闭")
			return
		}
	}
}

func initHandler(t *testing.T) *ali_deepseek.Handler {
	db := testioc.InitDB()
	err := dao.InitTables(db)
	require.NoError(t, err)
	configDao := dao.NewGORMConfigDAO(db)
	logDao := dao.NewGORMLLMLogDAO(db)
	configRepo := repository.NewCachedConfigRepository(configDao)
	logRepo := repository.NewLLMLogRepo(logDao)
	_, err = configDao.Save(context.Background(), dao.BizConfig{
		Id:             1,
		Biz:            "case",
		Price:          1,
		PromptTemplate: `请说一下%s天气`,
		Model:          "deepseek-r1",
	})
	require.NoError(t, err)
	return ali_deepseek.NewHandler("sk-1ff9e16afa654f50a0a9c759bd59274d", logRepo, configRepo)
}
