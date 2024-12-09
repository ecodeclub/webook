package integration

import (
	"context"
	"testing"

	"github.com/ecodeclub/webook/internal/ai/internal/domain"
	"github.com/ecodeclub/webook/internal/ai/internal/integration/startup"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/stretchr/testify/require"
)

func TestKnowledgeBaseTest(t *testing.T) {
	t.Skip("替换api.key运行")

	db := testioc.InitDB()
	startup.InitTableOnce(db)
	baseSvc := startup.InitKnowledgeBaseSvc(db, "api.key")
	testCases := []struct {
		name string
		file domain.KnowledgeBaseFile
	}{
		{
			name: "正常上传",
			file: domain.KnowledgeBaseFile{
				Biz:   "question",
				BizID: 4,
				Name:  "question4",
				Type:  domain.RepositoryBaseTypeRetrieval,
				Data:  []byte("test999999"),
				// 这个也要修改
				KnowledgeBaseID: "1863183941318684672",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 执行测试
			err := baseSvc.UploadFile(context.Background(), tc.file)
			require.NoError(t, err)
		})
	}

}
