//go:build e2e

package integration

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/ecodeclub/ekit/iox"
	"github.com/ecodeclub/webook/internal/email"
	emailmocks "github.com/ecodeclub/webook/internal/email/mocks"
	"github.com/ecodeclub/webook/internal/interview/internal/service"
	"github.com/ecodeclub/webook/internal/interview/internal/web"
	pdfmocks "github.com/ecodeclub/webook/internal/pkg/pdf/mocks"
	"github.com/ecodeclub/webook/internal/test"
	"github.com/gotomicro/ego/core/econf"
	"github.com/gotomicro/ego/server/egin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestOffer(t *testing.T) {
	ctrl := gomock.NewController(t)
	emailClient := emailmocks.NewMockService(ctrl)
	pdfClient := pdfmocks.NewMockConverter(ctrl)
	emailClient.EXPECT().SendMail(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, mail email.Mail) error {
		assert.Equal(t, mail, email.Mail{
			From:    "百度",
			To:      "john@example.com",
			Subject: fmt.Sprintf("【%s】%s岗位录取通知书", "百度", "后端"),
			Body: []byte(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8" />
    <title>百度 录用通知书</title>
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, "Noto Sans", "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif; background: #f7f8fa; margin: 0; padding: 24px; color: #1f2328; }
        .card { max-width: 720px; margin: 0 auto; background: #fff; border-radius: 12px; box-shadow: 0 6px 24px rgba(0,0,0,0.06); overflow: hidden; }
        .header { padding: 24px 28px; background: #0b5fff; color: #fff; }
        .header h1 { margin: 0; font-size: 22px; }
        .content { padding: 28px; line-height: 1.75; font-size: 15px; }
        .section { margin: 18px 0; }
        .kv { margin: 14px 0 0; padding: 0; list-style: none; }
        .kv li { margin: 8px 0; }
        .kv strong { display: inline-block; min-width: 90px; color: #555; }
        .note { background: #f8fafc; border: 1px solid #eef2f7; padding: 14px; border-radius: 8px; color: #475569; }
        .footer { padding: 20px 28px; border-top: 1px solid #f0f3f7; color: #6b7280; font-size: 13px; }
    </style>
</head>
<body>
<div class="card">
    <div class="header">
        <h1>百度 录用通知书</h1>
    </div>
    <div class="content">
        <div class="section">
            尊敬的候选人，<br />
            恭喜您通过面试评估！我们诚挚邀请您加入 <strong>百度</strong>，担任 <strong>后端</strong> 职位。
        </div>
        <div class="section">
            以下为本次录用的关键信息：
            <ul class="kv">
                <li><strong>岗位名称：</strong>后端</li>
                <li><strong>薪资待遇：</strong>12k</li>
                <li><strong>预计入职时间：</strong>2025年07月06日</li>
            </ul>
        </div>
        <div class="section note">
            请您在收到本通知后尽快与我们沟通入职安排。如对上述信息有任何疑问，欢迎随时联系 HR 与招聘团队。
        </div>
        <div class="section">
            我们期待与您携手共进，一同创造更大的价值！
        </div>
    </div>
    <div class="footer">
        此致<br />
        敬礼<br /><br />
        百度 招聘与人力资源团队
    </div>
</div>
</body>
</html>`),
			Attachments: []email.Attachment{
				{
					Filename: "岗位录取通知书.pdf",
					Content:  []byte("123"),
				},
			},
		})
		return nil
	}).AnyTimes()
	pdfClient.EXPECT().ConvertHTMLToPDF(gomock.Any(), gomock.Any()).Return([]byte("123"), nil).AnyTimes()

	svc := service.NewOfferService(emailClient, pdfClient)
	hdl := web.NewOfferHandler(svc)
	econf.Set("server", map[string]any{"contextTimeout": "1s"})
	server := egin.Load("server").Build()
	hdl.MemberRoutes(server.Engine)
	req, err := http.NewRequest(http.MethodPost,
		"/offer/send", iox.NewJSONReader(web.OfferSendRequest{
			Email:       "john@example.com",
			CompanyName: "百度",
			JobName:     "后端",
			Salary:      "12k",
			EntryTime:   1751801996000,
		}))
	require.NoError(t, err)
	req.Header.Set("content-type", "application/json")
	require.NoError(t, err)
	recorder := test.NewJSONResponseRecorder[int64]()
	server.ServeHTTP(recorder, req)
	require.Equal(t, 200, recorder.Code)
}
