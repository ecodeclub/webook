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
	t.Skip()
	handler := initHandler(t)
	msgChan, err := handler.StreamHandle(context.Background(), domain.LLMRequest{
		Biz:   "case",
		Uid:   23,
		Tid:   "tid1",
		Input: []string{"上海"},
	})
	require.NoError(t, err)
	// 修改后：
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
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
			log.Println(event.Content)
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
	})
	require.NoError(t, err)
	return ali_deepseek.NewHandler("you_key", logRepo, configRepo)
}
