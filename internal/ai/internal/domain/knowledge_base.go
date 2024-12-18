package domain

// 知识库的相关对象

// KnowledgeBaseFile 知识库文件
type KnowledgeBaseFile struct {
	Biz   string
	BizID int64
	// 平台
	Platform string
	// 文件名
	Name string
	// 文件内容
	Data []byte
	// 用途
	Type            string
	FileID          string
	KnowledgeBaseID string
}

const (
	RepositoryBaseTypeRetrieval = "retrieval"
	RepositoryBaseTypeFineTune  = "finetune"
)
