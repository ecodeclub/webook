package web

import (
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/webook/internal/cases"
	"github.com/ecodeclub/webook/internal/interactive"
	baguwen "github.com/ecodeclub/webook/internal/question"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	intrSvc     interactive.Service
	caseSvc     cases.Service
	caseSetSvc  cases.SetService
	caseExamSvc cases.ExamineService
	queSvc      baguwen.Service
	queSetSvc   baguwen.QuestionSetService
	queExamSvc  baguwen.ExamService
}

func NewHandler(
	intrSvc interactive.Service,
	caseSvc cases.Service,
	caseSetSvc cases.SetService,
	caseExamineSvc cases.ExamineService,
	queSvc baguwen.Service,
	queSetSvc baguwen.QuestionSetService,
	queExamSvc baguwen.ExamService,
) *Handler {
	return &Handler{
		intrSvc:     intrSvc,
		caseSvc:     caseSvc,
		queSvc:      queSvc,
		queSetSvc:   queSetSvc,
		queExamSvc:  queExamSvc,
		caseSetSvc:  caseSetSvc,
		caseExamSvc: caseExamineSvc,
	}
}

func (h *Handler) PrivateRoutes(server *gin.Engine) {
	g := server.Group("/interactive")
	g.POST("/collection/records", ginx.BS[CollectionInfoReq](h.CollectionRecords))
}
