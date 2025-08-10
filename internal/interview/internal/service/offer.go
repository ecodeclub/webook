package service

import (
	"bytes"
	"context"
	"fmt"
	"github.com/ecodeclub/webook/internal/pkg/pdf"
	"html/template"
	"time"

	"github.com/ecodeclub/webook/internal/email"
)

type OfferService interface {
	Send(ctx context.Context, req OfferSendReq) error
}

type OfferSendReq struct {
	EntryTime   int64  // 预计入职时间
	Salary      string // 前端拼接好 例如12k～18k啥的
	CompanyName string
	JobName     string
	ToEmail     string // 收件人邮箱
}

type offerService struct {
	emailClient  email.Service
	pdfConverter pdf.Converter
	template     string
}

func NewOfferService(
	emailClient email.Service,
	pdfConverter pdf.Converter,
	template string,
) OfferService {
	return &offerService{
		emailClient:  emailClient,
		pdfConverter: pdfConverter,
		template:     template,
	}
}

func (o *offerService) Send(ctx context.Context, req OfferSendReq) error {
	// 构造邮件主题
	subject := fmt.Sprintf("【%s】%s岗位录取通知书", req.CompanyName, req.JobName)

	// 构造邮件内容 (HTML格式)
	body, err := renderWithHTMLTemplate(o.template, req)
	if err != nil {
		return err
	}

	pdfByte, err := o.pdfConverter.ConvertHTMLToPDF(ctx, body)
	if err != nil {
		return err
	}
	// 构造邮件对象
	mail := email.Mail{
		From:    req.CompanyName,
		To:      req.ToEmail,
		Subject: subject,
		Body:    []byte(body),
		Attachments: []email.Attachment{
			{
				Filename: "岗位录取通知书.pdf",
				Content:  pdfByte,
			},
		},
	}

	// 发送邮件
	return o.emailClient.SendMail(ctx, mail)
}

type OfferData struct {
	CompanyName string
	JobName     string
	Salary      string
	EntryDate   string
}

func renderWithHTMLTemplate(tmpl string, req OfferSendReq) (string, error) {
	t, err := template.New("offer").Parse(tmpl)
	if err != nil {
		return "", err
	}
	data := OfferData{
		CompanyName: req.CompanyName,
		JobName:     req.JobName,
		Salary:      req.Salary,
		EntryDate:   time.Unix(req.EntryTime, 0).Format("2006年01月02日"),
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
