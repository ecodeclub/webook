package event

import (
	"encoding/json"
	"fmt"

	"github.com/ecodeclub/webook/internal/ai"
	"github.com/ecodeclub/webook/internal/question/internal/domain"
)

const (
	KnowledgeBaseUploadTopic = "knowledge_base_upload_topic"
)

type KnowledgeBaseEvent struct {
	Biz   string `json:"biz"`
	BizID int64  `json:"bizID"`
	// 文件名
	Name string `json:"name"`
	// 文件内容
	Data []byte `json:"data"`
	// 用途
	Type            string `json:"type"`
	KnowledgeBaseID string `json:"knowledgeBaseID"`
}

func NewKnowledgeBaseEvent(que domain.Question) (KnowledgeBaseEvent, error) {
	data, err := json.Marshal(que)
	if err != nil {
		return KnowledgeBaseEvent{}, fmt.Errorf("序列化问题数据失败 %w", err)
	}
	return KnowledgeBaseEvent{
		Biz:   domain.QuestionBiz,
		BizID: que.Id,
		Name:  fmt.Sprintf("question_%d", que.Id),
		Data:  data,
		Type:  ai.RepositoryBaseTypeRetrieval,
	}, nil
}
