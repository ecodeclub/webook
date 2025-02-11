package event

// KnowledgeBaseUploadEvent 知识库上传事件
type KnowledgeBaseUploadEvent struct {
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

const (
	KnowledgeBaseUploadTopic = "knowledge_base_upload_topic"
)
