package web

import (
	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/webook/internal/ai/internal/domain"
	"github.com/ecodeclub/webook/internal/ai/internal/service"
	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	svc service.ConfigService
}

func NewAdminHandler(svc service.ConfigService) *AdminHandler {
	return &AdminHandler{
		svc: svc,
	}
}

func (h *AdminHandler) RegisterRoutes(server *gin.Engine) {
	// 管理员路由组
	admin := server.Group("/ai/config")
	admin.POST("/save", ginx.B[ConfigRequest](h.Save))
	admin.GET("/list", ginx.W(h.List))
	admin.POST("/detail", ginx.B[ConfigInfoReq](h.GetById))
}

func (h *AdminHandler) Save(ctx *ginx.Context, req ConfigRequest) (ginx.Result, error) {
	id, err := h.svc.Save(ctx, domain.BizConfig{
		Id:             req.Config.Id,
		Biz:            req.Config.Biz,
		MaxInput:       req.Config.MaxInput,
		Model:          req.Config.Model,
		Price:          req.Config.Price,
		Temperature:    req.Config.Temperature,
		TopP:           req.Config.TopP,
		SystemPrompt:   req.Config.SystemPrompt,
		PromptTemplate: req.Config.PromptTemplate,
		KnowledgeId:    req.Config.KnowledgeId,
	})
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: id,
	}, nil
}

// List 获取配置列表
func (h *AdminHandler) List(ctx *ginx.Context) (ginx.Result, error) {
	configs, err := h.svc.List(ctx)
	if err != nil {
		return systemErrorResult, err
	}

	return ginx.Result{
		Data: slice.Map(configs, func(idx int, c domain.BizConfig) Config {
			return h.domainToConfig(c)
		}),
	}, nil
}

func (h *AdminHandler) GetById(ctx *ginx.Context, req ConfigInfoReq) (ginx.Result, error) {
	id := req.Id
	config, err := h.svc.GetById(ctx, id)
	if err != nil {
		return systemErrorResult, err
	}

	return ginx.Result{
		Data: h.domainToConfig(config),
	}, nil
}

func (h *AdminHandler) domainToConfig(cfg domain.BizConfig) Config {
	return Config{
		Id:             cfg.Id,
		Biz:            cfg.Biz,
		MaxInput:       cfg.MaxInput,
		Model:          cfg.Model,
		Price:          cfg.Price,
		Temperature:    cfg.Temperature,
		TopP:           cfg.TopP,
		SystemPrompt:   cfg.SystemPrompt,
		PromptTemplate: cfg.PromptTemplate,
		KnowledgeId:    cfg.KnowledgeId,
		Utime:          cfg.Utime,
	}
}
