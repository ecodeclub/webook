package simple

import (
	"github.com/ecodeclub/webook/internal/ai/internal/service/handler"
	"github.com/ecodeclub/webook/internal/ai/internal/service/handler/credit"
	"github.com/ecodeclub/webook/internal/ai/internal/service/handler/gpt"
	"github.com/ecodeclub/webook/internal/ai/internal/service/handler/log"
)

func InitHandler(
	logHandler *log.Handler,
	creditHandler *credit.Handler,
	gptHandler *gpt.Handler,
) *Handler {
	handlers := []handler.GptHandler{
		logHandler,
		creditHandler,
		gptHandler,
	}
	var h handler.HandleFunc
	for i := len(handlers) - 1; i >= 0; i-- {
		h = handlers[i].Next(h)
	}
	return &Handler{
		handlerFunc: h,
	}
}
