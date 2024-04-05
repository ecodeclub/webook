package ioc

import (
	"fmt"
	"net/http"
	"time"

	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/member"
)

type MembershipChecker struct {
	svc member.Service
}

func (c *MembershipChecker) Membership(ctx *ginx.Context, sess session.Session) (ginx.Result, error) {
	claims := sess.Claims()

	memberDDL, err := time.Parse(time.DateTime, claims.Get("memberDDL").StringOrDefault(""))
	if err == nil {
		// 找到会员截止日期
		if memberDDL.Local().Compare(time.Now().Local()) <= 0 {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			// todo: 替换为 ginx.ErrUnauthorized
			return ginx.Result{}, fmt.Errorf("会员已过期 uid: %d", claims.Uid)
		}
		return ginx.Result{}, nil
	}

	// 未找到会员截止日期
	// 查询svc
	info, err := c.svc.GetMembershipInfo(ctx.Request.Context(), claims.Uid)
	if err != nil {
		ctx.AbortWithStatus(http.StatusUnauthorized)
		// todo: 替换为 ginx.ErrUnauthorized
		return ginx.Result{}, fmt.Errorf("获取会员信息失败 uid: %d", claims.Uid)
	}

	// 再原有jwt数据中添加会员截止日期
	jwtData := claims.Data
	jwtData["memberDDL"] = time.Unix(info.EndAt, 0).Local().Format(time.DateTime)

	// 刷新session
	_, err = session.NewSessionBuilder(ctx, claims.Uid).SetJwtData(jwtData).Build()
	if err != nil {
		ctx.AbortWithStatus(http.StatusUnauthorized)
		// todo: 替换为 ginx.ErrUnauthorized
		return ginx.Result{}, fmt.Errorf("生成新session失败 uid: %d", claims.Uid)
	}

	return ginx.Result{}, nil
}
