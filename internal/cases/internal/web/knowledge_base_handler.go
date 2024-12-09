package web

import (
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/webook/internal/cases/internal/service"
	"github.com/gin-gonic/gin"
)

type KnowledgeBaseHandler struct {
	knowledgeSvc service.KnowledgeBaseService
}

func NewKnowledgeBaseHandler(svc service.KnowledgeBaseService) *KnowledgeBaseHandler {
	return &KnowledgeBaseHandler{
		knowledgeSvc: svc,
	}
}

func (h *KnowledgeBaseHandler) PrivateRoutes(server *gin.Engine) {
	server.GET("/case/knowledgeBase/syncAll", ginx.W(h.SyncAll))
}

func (h *KnowledgeBaseHandler) SyncAll(ctx *ginx.Context) (ginx.Result, error) {
	// 异步 上传全量问题到ai的知识库。
	go h.knowledgeSvc.FullSync()
	return ginx.Result{}, nil
}
