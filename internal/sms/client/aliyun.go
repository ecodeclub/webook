package client

import (
	"encoding/json"
	"fmt"
	"strings"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dysmsapi "github.com/alibabacloud-go/dysmsapi-20170525/v4/client"
	"github.com/alibabacloud-go/tea/tea"
)

const signName = "妙影科技"

var (
	// platformTemplateType2Aliyun  平台内部模版状态到阿里云状态的映射
	platformTemplateType2Aliyun = map[TemplateType]TemplateType{
		TemplateTypeVerification:  TemplateTypeInternational,
		TemplateTypeNotification:  TemplateTypeMarketing,
		TemplateTypeMarketing:     TemplateTypeNotification,
		TemplateTypeInternational: TemplateTypeVerification,
	}
	_ Client = (*AliyunSMS)(nil)
)

// AliyunSMS 阿里云短信实现
type AliyunSMS struct {
	client *dysmsapi.Client
}

// NewAliyunSMS 创建阿里云短信实例
func NewAliyunSMS(accessKeyID, accessKeySecret string) (*AliyunSMS, error) {
	config := &openapi.Config{
		AccessKeyId:     tea.String(accessKeyID),
		AccessKeySecret: tea.String(accessKeySecret),
		Endpoint:        tea.String("dysmsapi.aliyuncs.com"),
	}

	client, err := dysmsapi.NewClient(config)
	if err != nil {
		return nil, err
	}
	return &AliyunSMS{client: client}, nil
}

func (a *AliyunSMS) Send(req SendReq) (SendResp, error) {
	if len(req.PhoneNumbers) == 0 {
		return SendResp{}, fmt.Errorf("%w: %v", ErrInvalidParameter, "手机号码不能为空")
	}

	// 将多个手机号码用逗号分隔
	phoneNumbers := ""
	for i, phone := range req.PhoneNumbers {
		if i > 0 {
			phoneNumbers += ","
		}
		phoneNumbers += phone
	}

	templateParam := ""
	if req.TemplateParam != nil {
		jsonParams, err := json.Marshal(req.TemplateParam)
		if err != nil {
			return SendResp{}, fmt.Errorf("%w: %w", ErrInvalidParameter, err)
		}
		templateParam = string(jsonParams)
	}

	request := &dysmsapi.SendSmsRequest{
		PhoneNumbers:  tea.String(phoneNumbers),
		SignName:      tea.String(signName),
		TemplateCode:  tea.String(req.TemplateID),
		TemplateParam: tea.String(templateParam),
	}

	response, err := a.client.SendSms(request)
	if err != nil {
		return SendResp{}, fmt.Errorf("%w: %w", ErrSendFailed, err)
	}

	if response.Body == nil || response.Body.Code == nil || *response.Body.Code != OK {
		return SendResp{}, fmt.Errorf("%w: %v", ErrSendFailed, "响应异常")
	}

	// 构建新的响应格式
	result := SendResp{
		RequestID:    *response.Body.RequestId,
		PhoneNumbers: make(map[string]SendRespStatus),
	}

	// 阿里云短信发送接口不返回每个手机号的状态，只返回整体状态
	// 所以这里为每个手机号设置相同的状态
	for _, phone := range req.PhoneNumbers {
		// 去掉可能的+86前缀
		cleanPhone := strings.TrimPrefix(phone, "+86")
		result.PhoneNumbers[cleanPhone] = SendRespStatus{
			Code:    *response.Body.Code,
			Message: *response.Body.Message,
		}
	}
	return result, nil
}
