package client

import (
	"errors"
)

const (
	OK = "Ok"
)

// 通用错误定义
var (
	ErrCreateTemplateFailed = errors.New("创建模版失败")
	ErrQueryTemplateStatus  = errors.New("查询模版状态失败")
	ErrSendFailed           = errors.New("发送短信失败")
	ErrQuerySendDetails     = errors.New("查询发送详情失败")
	ErrInvalidParameter     = errors.New("参数无效")
)

type (
	AuditStatus  int
	TemplateType int32
)

type SendStatus int

const (
	TemplateTypeInternational TemplateType = 0 // 国际/港澳台消息 仅阿里云使用
	TemplateTypeMarketing     TemplateType = 1 // 营销短信
	TemplateTypeNotification  TemplateType = 2 // 通知短信
	TemplateTypeVerification  TemplateType = 3 // 验证码

	AuditStatusPending  AuditStatus = 0 // 审核中
	AuditStatusApproved AuditStatus = 1 // 审核通过
	AuditStatusRejected AuditStatus = 2 // 审核拒绝

	SendStatusWaiting SendStatus = 1 // 等待回执
	SendStatusSuccess SendStatus = 3 // 发送成功
	SendStatusFailed  SendStatus = 2 // 发送失败
)

// Client 短信客户端接口 (抽象)
//
//go:generate mockgen -source=./types.go -destination=./mocks/sms.mock.go -package=smsmocks -typed Client
type Client interface {
	// Send 发送短信
	Send(req SendReq) (SendResp, error)
}

// SendReq 发送短信请求参数
type SendReq struct {
	PhoneNumbers []string // 手机号码, 阿里云、腾讯云共用
	// SignName      string            // 签名名称, 阿里云、腾讯云共用
	TemplateID    string            // 模板 ID, 阿里云、腾讯云共用
	TemplateParam map[string]string // 模板参数, 阿里云、腾讯云共用, key-value 形式 // date
}

// SendResp 发送短信响应参数
type SendResp struct {
	RequestID    string                    // 请求 ID,      阿里云、腾讯云共用
	PhoneNumbers map[string]SendRespStatus // 去掉+86后的手机号
}

type SendRespStatus struct {
	Code    string
	Message string
}
