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

	"github.com/ecodeclub/webook/internal/roadmap"

	"github.com/ecodeclub/webook/internal/search"

	"github.com/ecodeclub/ginx/middlewares/activelimit/locallimit"
	"github.com/ecodeclub/webook/internal/interactive"

	"github.com/ecodeclub/webook/internal/credit"
	"github.com/ecodeclub/webook/internal/marketing"
	"github.com/ecodeclub/webook/internal/payment"
	"github.com/ecodeclub/webook/internal/project"

	"github.com/ecodeclub/webook/internal/feedback"
	"github.com/ecodeclub/webook/internal/order"
	"github.com/ecodeclub/webook/internal/product"

	"github.com/ecodeclub/webook/internal/pkg/middleware"
	"github.com/ecodeclub/webook/internal/skill"

	"github.com/ecodeclub/webook-private/nonsense"

	"github.com/ecodeclub/webook/internal/cases"

	"github.com/ecodeclub/webook/internal/label"

	"github.com/ecodeclub/webook/internal/cos"

	baguwen "github.com/ecodeclub/webook/internal/question"

	"github.com/gin-gonic/gin"

	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/user"
	"github.com/gin-contrib/cors"
	"github.com/gotomicro/ego/server/egin"
)

func initGinxServer(sp session.Provider,
	checkMembershipMiddleware *middleware.CheckMembershipMiddlewareBuilder,
	localActiveLimiterMiddleware *locallimit.LocalActiveLimit,
	// 这个暂时用不上
	checkPermissionMiddleware *middleware.CheckPermissionMiddlewareBuilder,
	qh *baguwen.Handler,
	examineHdl *baguwen.ExamineHandler,
	qsh *baguwen.QuestionSetHandler,
	lhdl *label.Handler,
	user *user.Handler,
	cosHdl *cos.Handler,
	caseHdl *cases.Handler,
	skillHdl *skill.Handler,
	fbHdl *feedback.Handler,
	pHdl *product.Handler,
	orderHdl *order.Handler,
	prjHdl *project.Handler,
	creditHdl *credit.Handler,
	paymentHdl *payment.Handler,
	marketingHdl *marketing.Handler,
	intrHdl *interactive.Handler,
	searchHdl *search.Handler,
	roadmapHdl *roadmap.Handler,
) *egin.Component {
	session.SetDefaultProvider(sp)
	res := egin.Load("web").Build()
	// 基本的含义就是执行方法的时候优先考虑 gin.Context，而后考虑 gin.Request.Context
	res.Engine.ContextWithFallback = true
	res.Use(cors.New(cors.Config{
		ExposeHeaders:    []string{"X-Refresh-Token", "X-Access-Token"},
		AllowCredentials: true,
		AllowHeaders:     []string{"X-Timestamp", "Authorization", "Content-Type"},
		AllowOriginFunc: func(origin string) bool {
			if strings.HasPrefix(origin, "http://localhost") {
				return true
			}
			// 只允许我的域名过来的
			return strings.Contains(origin, "meoying.com")
		},
	}))
	res.GET("/hello", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "hello, world!")
	})

	// 微信支付的回调不需要安全校验机制
	paymentHdl.PublicRoutes(res.Engine)

	// 虽然叫做 NonSense，但是我还是得告诉你，这是一个安全校验机制
	// 但是我并不能在开源里面放出来，因为知道了如何校验，就知道了如何破解
	// 虽然理论上可以用 plugin 机制，但是 plugin 机制比较容易遇到不兼容的问题
	// 实在不想处理
	res.Use(nonsense.NonSenseV1)
	res.Use(localActiveLimiterMiddleware.Build())
	user.PublicRoutes(res.Engine)
	qh.PublicRoutes(res.Engine)
	cosHdl.PublicRoutes(res.Engine)
	caseHdl.PublicRoutes(res.Engine)
	skillHdl.PublicRoutes(res.Engine)
	// 登录校验
	res.Use(session.CheckLoginMiddleware())
	user.PrivateRoutes(res.Engine)
	lhdl.PrivateRoutes(res.Engine)
	qsh.PrivateRoutes(res.Engine)
	cosHdl.PrivateRoutes(res.Engine)
	caseHdl.PrivateRoutes(res.Engine)
	skillHdl.PrivateRoutes(res.Engine)
	pHdl.PrivateRoutes(res.Engine)
	orderHdl.PrivateRoutes(res.Engine)
	searchHdl.PrivateRoutes(res.Engine)
	roadmapHdl.PrivateRoutes(res.Engine)

	creditHdl.PrivateRoutes(res.Engine)
	marketingHdl.PrivateRoutes(res.Engine)
	intrHdl.PrivateRoutes(res.Engine)

	// 权限校验
	prjHdl.PrivateRoutes(res.Engine)

	// 会员校验
	res.Use(checkMembershipMiddleware.Build())
	qh.MemberRoutes(res.Engine)
	examineHdl.MemberRoutes(res.Engine)
	caseHdl.MemberRoutes(res.Engine)
	fbHdl.MemberRoutes(res.Engine)
	return res
}
