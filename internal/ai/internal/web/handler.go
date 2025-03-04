package web

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/ai/internal/domain"
	"github.com/ecodeclub/webook/internal/ai/internal/errs"
	"github.com/ecodeclub/webook/internal/ai/internal/service"
	"github.com/ecodeclub/webook/internal/ai/internal/service/llm/handler/credit"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

type Handler struct {
	generalSvc service.GeneralService
	jdSvc      service.JDService
}

func NewHandler(generalSvc service.GeneralService, jdSvc service.JDService) *Handler {
	return &Handler{
		generalSvc: generalSvc,
		jdSvc:      jdSvc,
	}
}

func (h *Handler) MemberRoutes(server *gin.Engine) {
	server.POST("/ai/ask", ginx.BS(h.LLMAsk))
	server.POST("/ai/analysis_jd", ginx.BS(h.AnalysisJd))
	server.POST("/ai/stream", h.Stream)

}

func (h *Handler) LLMAsk(ctx *ginx.Context, req LLMRequest, sess session.Session) (ginx.Result, error) {
	uid := sess.Claims().Uid
	resp, err := h.generalSvc.LLMAsk(ctx, uid, req.Biz, req.Input)
	switch {
	case errors.Is(err, credit.ErrInsufficientCredit):
		return ginx.Result{
			Code: errs.InsufficientCredit.Code,
			Msg:  errs.InsufficientCredit.Msg,
		}, nil
	case err == nil:
		return ginx.Result{
			Data: LLMResponse{
				Amount:    resp.Amount,
				RawResult: resp.Answer,
			},
		}, nil
	default:
		return systemErrorResult, err
	}
}

func (h *Handler) AnalysisJd(ctx *ginx.Context, req JDRequest, sess session.Session) (ginx.Result, error) {
	uid := sess.Claims().Uid
	resp, err := h.jdSvc.Evaluate(ctx, uid, req.JD)
	switch {
	case errors.Is(err, credit.ErrInsufficientCredit):
		return ginx.Result{
			Code: errs.InsufficientCredit.Code,
			Msg:  errs.InsufficientCredit.Msg,
		}, nil
	case err == nil:
		return ginx.Result{
			Data: JDResponse{
				Amount:    resp.Amount,
				TechScore: h.newJD(resp.TechScore),
				BizScore:  h.newJD(resp.BizScore),
				PosScore:  h.newJD(resp.PosScore),
				Subtext:   resp.Subtext,
			},
		}, nil
	default:
		return systemErrorResult, err
	}

}

func (h *Handler) Stream(ctx *gin.Context) {
	// 设置 SSE 响应头
	ctx.Writer.Header().Set("Content-Type", "text/event-stream")
	ctx.Writer.Header().Set("Cache-Control", "no-cache")
	ctx.Writer.Header().Set("Connection", "keep-alive")

	gtx := &ginx.Context{Context: ctx}
	sess, err := session.Get(gtx)
	if err != nil {
		slog.Debug("获取 Session 失败", slog.Any("err", err))
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	uid := sess.Claims().Uid
	var req LLMRequest
	// Bind 方法本身会返回 400 的错误
	if err := ctx.Bind(&req); err != nil {
		slog.Debug("绑定参数失败", slog.Any("err", err))
		return
	}
	timeCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	ch, err := h.generalSvc.Stream(timeCtx, uid, req.Biz, req.Input)
	cancel()
	if err != nil {
		h.chatErr(ctx, err)
		return
	}
	h.stream(ctx, ch)
}

func (h *Handler) chatErr(ctx *gin.Context, err error) {
	evt := Event{
		Type: ErrEvt,
		Err:  err.Error(),
	}
	evtStr, _ := json.Marshal(evt)
	sendEvent(ctx, string(evtStr))
}

func (h *Handler) chatMsg(ctx *gin.Context, domainEvt domain.StreamEvent) {
	evt := Event{
		Type: MsgEvt,
		Data: EvtMsg{
			Content:          domainEvt.Content,
			ReasoningContent: domainEvt.ReasoningContent,
		},
	}
	evtStr, _ := json.Marshal(evt)
	sendEvent(ctx, string(evtStr))
}

func (h *Handler) chatEnd(ctx *gin.Context) {
	evt := Event{
		Type: EndEvt,
	}
	evtStr, _ := json.Marshal(evt)
	sendEvent(ctx, string(evtStr))
}

func (h *Handler) stream(ctx *gin.Context, ch chan domain.StreamEvent) {

	for {
		select {
		case event, ok := <-ch:
			if !ok || event.Done {
				// 通道关闭，发送结束事件
				h.chatEnd(ctx)
				return
			}

			if event.Error != nil {
				h.chatErr(ctx, event.Error)
				return
			}
			h.chatMsg(ctx, event)
		case <-ctx.Request.Context().Done():
			return
		}
	}
}

func sendEvent(ctx *gin.Context, data string) {
	buf := bytes.Buffer{}
	buf.WriteString(data)
	buf.WriteByte('\n')
	_, _ = ctx.Writer.Write(buf.Bytes())
	ctx.Writer.Flush()
}

func (h *Handler) newJD(jd domain.JDEvaluation) JDEvaluation {
	return JDEvaluation{
		Score:    jd.Score,
		Analysis: jd.Analysis,
	}
}
