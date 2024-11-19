// Copyright 2023 ecodeclub
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ai

import (
	"github.com/ecodeclub/webook/internal/ai/internal/service/llm/handler"
	"github.com/ecodeclub/webook/internal/ai/internal/service/llm/handler/biz"
	"github.com/ecodeclub/webook/internal/ai/internal/service/llm/handler/config"
	"github.com/ecodeclub/webook/internal/ai/internal/service/llm/handler/credit"
	"github.com/ecodeclub/webook/internal/ai/internal/service/llm/handler/log"
	"github.com/ecodeclub/webook/internal/ai/internal/service/llm/handler/platform/zhipu"
	"github.com/ecodeclub/webook/internal/ai/internal/service/llm/handler/record"
	"github.com/gotomicro/ego/core/econf"
)

func InitHandlerFacade(common []handler.Builder,
	zhipu *zhipu.Handler) *biz.FacadeHandler {
	que := InitQuestionExamineHandler(common, zhipu)
	c := InitCaseExamineHandler(common, zhipu)
	jdTech := InitJDTechHandler(common, zhipu)
	jdBiz := InitJDBizHandler(common, zhipu)
	jdPosition := InitJDPositionHandler(common, zhipu)
	return biz.NewHandler(map[string]handler.Handler{
		que.Biz():        que,
		c.Biz():          c,
		jdBiz.Biz():      jdBiz,
		jdTech.Biz():     jdTech,
		jdPosition.Biz(): jdPosition,
	})
}

func InitZhipu() *zhipu.Handler {
	type Config struct {
		APIKey string `yaml:"apikey"`
	}
	var cfg Config
	err := econf.UnmarshalKey("zhipu", &cfg)
	if err != nil {
		panic(err)
	}
	h, err := zhipu.NewHandler(cfg.APIKey)
	if err != nil {
		panic(err)
	}
	return h
}

func InitQuestionExamineHandler(
	common []handler.Builder,
	// platform 就是真正的出口
	platform handler.Handler) *biz.CompositionHandler {
	// log -> cfg -> credit -> record -> question_examine -> platform
	builder := biz.NewQuestionExamineBizHandlerBuilder()
	common = append(common, builder)
	return biz.NewCombinedBizHandler("question_examine", common, platform)

}

func InitCaseExamineHandler(
	common []handler.Builder,
	// platform 就是真正的出口
	platform handler.Handler) *biz.CompositionHandler {
	builder := biz.NewCaseExamineBizHandlerBuilder()
	common = append(common, builder)
	return biz.NewCombinedBizHandler("case_examine", common, platform)
}

func InitJDTechHandler(common []handler.Builder,
	platform handler.Handler) *biz.CompositionHandler {
	builder := biz.NewJDTechHandlerBuilder()
	common = append(common, builder)
	return biz.NewCombinedBizHandler("analysis_jd_tech", common, platform)
}
func InitJDBizHandler(common []handler.Builder,
	platform handler.Handler) *biz.CompositionHandler {
	builder := biz.NewJDBizHandlerBuilder()
	common = append(common, builder)
	return biz.NewCombinedBizHandler("analysis_jd_biz", common, platform)
}
func InitJDPositionHandler(common []handler.Builder,
	platform handler.Handler) *biz.CompositionHandler {
	builder := biz.NewJDPositionHandlerBuilder()
	common = append(common, builder)
	return biz.NewCombinedBizHandler("analysis_jd_position", common, platform)
}
func InitCommonHandlers(log *log.HandlerBuilder,
	cfg *config.HandlerBuilder,
	credit *credit.HandlerBuilder,
	record *record.HandlerBuilder) []handler.Builder {
	return []handler.Builder{log, cfg, credit, record}
}
