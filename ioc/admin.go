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

package ioc

import (
	"net/http"
	"strings"

	"github.com/ecodeclub/webook/internal/review"

	"github.com/ecodeclub/webook/internal/ai"

	"github.com/ecodeclub/webook/internal/cases"

	baguwen "github.com/ecodeclub/webook/internal/question"

	"github.com/ecodeclub/webook/internal/roadmap"

	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/marketing"
	"github.com/ecodeclub/webook/internal/project"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/elog"
	"github.com/gotomicro/ego/server/egin"
)

type AdminServer *egin.Component

func InitAdminServer(prj *project.AdminHandler,
	rm *roadmap.AdminHandler,
	que *baguwen.AdminHandler,
	queSet *baguwen.AdminQuestionSetHandler,
	caseHdl *cases.AdminCaseHandler,
	caseSetHdl *cases.AdminCaseSetHandler,
	mark *marketing.AdminHandler,
	aiHdl *ai.AdminHandler,
	reviewAdminHdl *review.AdminHdl,
	caseKnowledgeBaseHdl *cases.KnowledgeBaseHandler,
	queKnowledgeBaseHdl *baguwen.KnowledgeBaseHandler,
) AdminServer {
	res := egin.Load("admin").Build()
	res.Use(cors.New(cors.Config{
		ExposeHeaders:    []string{"X-Refresh-Token", "X-Access-Token"},
		AllowCredentials: true,
		AllowHeaders:     []string{"X-Timestamp", "Authorization", "Content-Type"},
		AllowOriginFunc: func(origin string) bool {
			if strings.HasPrefix(origin, "http://localhost") {
				return true
			}
			// 只允许我的域名过来的
			return strings.Contains(origin, "meoying.com") ||
				strings.Contains(origin, "mianshi.icu")
		},
	}))
	res.GET("/hello", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "hello, world!")
	})

	// 安全校验
	//res.Use(nonsense.NonSenseV1)
	// 登录校验
	res.Use(session.CheckLoginMiddleware())
	res.Use(AdminPermission())
	prj.PrivateRoutes(res.Engine)
	queSet.PrivateRoutes(res.Engine)
	mark.PrivateRoutes(res.Engine)
	rm.PrivateRoutes(res.Engine)
	que.PrivateRoutes(res.Engine)
	caseHdl.PrivateRoutes(res.Engine)
	caseSetHdl.PrivateRoutes(res.Engine)
	aiHdl.RegisterRoutes(res.Engine)
	reviewAdminHdl.PrivateRoutes(res.Engine)
	queKnowledgeBaseHdl.PrivateRoutes(res.Engine)
	caseKnowledgeBaseHdl.PrivateRoutes(res.Engine)
	return res
}

func AdminPermission() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		xctx := &ginx.Context{Context: ctx}
		sess, err := session.Get(xctx)
		if err != nil {
			ctx.AbortWithStatus(http.StatusInternalServerError)
			elog.Error("非法访问 admin 接口", elog.FieldErr(err))
			return
		}
		if sess.Claims().Get("creator").StringOrDefault("") != "true" {
			ctx.AbortWithStatus(http.StatusInternalServerError)
			elog.Error("非法访问 admin 接口，未设置权限")
			return
		}
	}
}
