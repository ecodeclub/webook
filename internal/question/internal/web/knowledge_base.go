package web

import (
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/webook/internal/question/internal/service"
	"github.com/gin-gonic/gin"
)

type KnowledgeBaseHandler struct {
	knowledgeSvc service.QuestionKnowledgeBase
}

func NewKnowledgeBaseHandler(svc service.QuestionKnowledgeBase) *KnowledgeBaseHandler {
	return &KnowledgeBaseHandler{
		knowledgeSvc: svc,
	}
}

func (h *KnowledgeBaseHandler) PrivateRoutes(server *gin.Engine) {
	server.GET("/question/knowledgeBase/syncAll", ginx.W(h.SyncAll))
}

func (h *KnowledgeBaseHandler) SyncAll(ctx *ginx.Context) (ginx.Result, error) {
	// 异步 上传全量问题到ai的知识库。
	go h.knowledgeSvc.FullSync()
	return ginx.Result{}, nil
}
