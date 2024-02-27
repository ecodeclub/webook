package web

import (
	"fmt"
	"log"
	"time"

	"github.com/ecodeclub/ekit/bean/copier"
	"github.com/ecodeclub/ekit/bean/copier/converter"
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/question/internal/domain"
	"github.com/ecodeclub/webook/internal/question/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/elog"
)

type QuestionSetHandler struct {
	vo2dm          copier.Copier[Question, domain.Question]
	dm2vo          copier.Copier[domain.Question, Question]
	svc            service.Service
	questionSetSvc service.QuestionSetService
	logger         *elog.Component
}

func NewQuestionSetHandler(qss service.QuestionSetService) (*QuestionSetHandler, error) {
	vo2dm, err := copier.NewReflectCopier[Question, domain.Question](
		copier.IgnoreFields("Utime"),
	)
	if err != nil {
		return nil, err
	}
	cnvter := converter.ConverterFunc[time.Time, string](func(src time.Time) (string, error) {
		return src.Format(time.DateTime), nil
	})
	dm2vo, err := copier.NewReflectCopier[domain.Question, Question](
		copier.ConvertField[time.Time, string]("Utime", cnvter),
	)
	if err != nil {
		return nil, err
	}
	return &QuestionSetHandler{
		vo2dm:          vo2dm,
		dm2vo:          dm2vo,
		questionSetSvc: qss,
		logger:         elog.DefaultLogger,
	}, nil
}

func (h *QuestionSetHandler) PrivateRoutes(server *gin.Engine) {

	server.POST("/question-sets/create", ginx.BS[CreateQuestionSetReq](h.CreateQuestionSet))
	// 题集更新接口 覆盖式的 前端传递题集中最终的问题集合
	server.POST("/question-sets/update", ginx.BS[UpdateQuestionsOfQuestionSetReq](h.UpdateQuestionsOfQuestionSet))

	// 查找题集，分页接口，只有分页参数，不需要传递 UserID
	server.POST("/question-sets/list", ginx.BS[Page](h.ListQuestionSet))

	// 题集详情：标题，描述，题目（题目暂时不分页，一个题集不会有很多）。题目包含适合展示在列表上的字段
	server.POST("/question-sets/detail", ginx.BS[QuestionSetID](h.RetrieveQuestionSetDetail))
}

// CreateQuestionSet 创建题集
func (h *QuestionSetHandler) CreateQuestionSet(ctx *ginx.Context, req CreateQuestionSetReq, sess session.Session) (ginx.Result, error) {
	id, err := h.questionSetSvc.Create(ctx, domain.QuestionSet{Uid: sess.Claims().Uid, Title: req.Title, Description: req.Description})
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: id,
	}, nil
}

func (h *QuestionSetHandler) UpdateQuestionsOfQuestionSet(ctx *ginx.Context, req UpdateQuestionsOfQuestionSetReq, sess session.Session) (ginx.Result, error) {
	questions := make([]domain.Question, len(req.QIDs))
	for i := range req.QIDs {
		questions[i] = domain.Question{Id: req.QIDs[i]}
	}
	log.Printf("questions = %#v\n", questions)

	// todo: 验证题集ID是否属于当前用户

	err := h.questionSetSvc.UpdateQuestions(ctx.Request.Context(), domain.QuestionSet{
		Id:        req.QSID,
		Questions: questions,
	})
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{}, nil
}

func (h *QuestionSetHandler) ListQuestionSet(ctx *ginx.Context, req Page, sess session.Session) (ginx.Result, error) {
	// todo: 未实现
	// 制作库不需要统计总数
	data, cnt, err := h.svc.List(ctx, req.Offset, req.Limit, sess.Claims().Uid)
	if err != nil {
		return systemErrorResult, err
	}
	fmt.Println(cnt)
	return ginx.Result{
		Data: data,
	}, nil
}

func (h *QuestionSetHandler) RetrieveQuestionSetDetail(ctx *ginx.Context, req QuestionSetID, sess session.Session) (ginx.Result, error) {
	// todo: 未实现
	detail, err := h.svc.PubDetail(ctx, req.QuestionSetID)
	if err != nil {
		return systemErrorResult, err
	}
	vo, err := h.dm2vo.Copy(&detail)
	return ginx.Result{
		Data: vo,
	}, err
}
