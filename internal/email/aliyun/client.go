package aliyun

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dm20151123 "github.com/alibabacloud-go/dm-20151123/v2/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	credential "github.com/aliyun/credentials-go/credentials"

	"github.com/ecodeclub/webook/internal/email"
)

// AliyunDirectMailAPI 阿里云邮件推送API客户端
type AliyunDirectMailAPI struct {
	client    *dm20151123.Client
	fromEmail string
}

// NewAliyunDirectMailAPI 创建阿里云邮件推送API客户端
// accessKeyID: Access Key ID
// accessKeySecret: Access Key Secret
// fromEmail: 发信地址，例如 noreply@mailer.meoying.com
// accountName: 发信人昵称
// region: 地域，例如 "cn-hangzhou"
func NewAliyunDirectMailAPI(accessKeyID, accessKeySecret, accountName string) (*AliyunDirectMailAPI, error) {
	// 使用AccessKey方式创建凭据
	config := &credential.Config{
		Type:            tea.String("access_key"),
		AccessKeyId:     tea.String(accessKeyID),
		AccessKeySecret: tea.String(accessKeySecret),
	}

	cred, err := credential.NewCredential(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create credential: %w", err)
	}

	// 创建API配置
	apiConfig := &openapi.Config{
		Credential: cred,
		Endpoint:   tea.String("dm.aliyuncs.com"),
	}

	// 创建客户端
	client, err := dm20151123.NewClient(apiConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create DirectMail client: %w", err)
	}

	return &AliyunDirectMailAPI{
		client:    client,
		fromEmail: accountName,
	}, nil
}

// SendMail 实现email.Service接口
func (a *AliyunDirectMailAPI) SendMail(ctx context.Context, mail email.Mail) error {
	// 构建发送邮件请求
	request := &dm20151123.SingleSendMailAdvanceRequest{
		AccountName:    tea.String(a.fromEmail),
		FromAlias:      tea.String(mail.From),
		AddressType:    tea.Int32(1), // 1表示随机账号
		ToAddress:      tea.String(mail.To),
		Subject:        tea.String(mail.Subject),
		HtmlBody:       tea.String(string(mail.Body)),
		ReplyToAddress: tea.Bool(false),
	}
	// 运行时选项
	runtime := &util.RuntimeOptions{}
	if len(mail.Attachments) > 0 {
		attachments := make([]*dm20151123.SingleSendMailAdvanceRequestAttachments, 0, len(mail.Attachments))
		for idx := range mail.Attachments {
			attachment := &mail.Attachments[idx]
			att := &dm20151123.SingleSendMailAdvanceRequestAttachments{}
			att.SetAttachmentName(attachment.Filename)
			att.SetAttachmentUrlObject(bytes.NewReader(attachment.Content))
			attachments = append(attachments, att)
		}
		request.Attachments = attachments
	}
	// 发送邮件
	_, err := a.client.SingleSendMailAdvance(request, runtime)
	if err != nil {
		return a.handleError(err)
	}

	return nil
}

// handleError 处理阿里云API错误
func (a *AliyunDirectMailAPI) handleError(err error) error {
	if sdkError, ok := err.(*tea.SDKError); ok {
		// 解析错误信息
		var errorData interface{}
		if sdkError.Data != nil {
			decoder := json.NewDecoder(strings.NewReader(tea.StringValue(sdkError.Data)))
			_ = decoder.Decode(&errorData)
		}

		// 构建详细错误信息
		errorMsg := fmt.Sprintf("阿里云邮件推送API错误: %s", tea.StringValue(sdkError.Message))

		if errorData != nil {
			if dataMap, ok := errorData.(map[string]interface{}); ok {
				if recommend, exists := dataMap["Recommend"]; exists {
					errorMsg += fmt.Sprintf(" | 建议: %v", recommend)
				}
				if requestId, exists := dataMap["RequestId"]; exists {
					errorMsg += fmt.Sprintf(" | RequestId: %v", requestId)
				}
			}
		}

		return errors.New(errorMsg)
	}

	return fmt.Errorf("邮件发送失败: %w", err)
}
