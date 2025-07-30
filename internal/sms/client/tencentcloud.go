package client

import (
	"fmt"
	"strings"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	sms "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sms/v20210111"
)

// TencentCloudSMS 腾讯云短信实现
type TencentCloudSMS struct {
	client *sms.Client
	appID  *string // 短信 appID
}

// NewTencentCloudSMS 创建腾讯云短信客户端
func NewTencentCloudSMS(regionID, secretID, secretKey, appID string) (*TencentCloudSMS, error) {
	client, err := sms.NewClient(common.NewCredential(secretID, secretKey), regionID, profile.NewClientProfile()) // 区域
	if err != nil {
		return nil, err
	}
	appIDPtr := &appID
	return &TencentCloudSMS{client: client, appID: appIDPtr}, nil
}

func (t *TencentCloudSMS) Send(req SendReq) (SendResp, error) {
	// https://cloud.tencent.com/document/product/382/55981
	if len(req.PhoneNumbers) == 0 {
		return SendResp{}, fmt.Errorf("%w: 手机号码不能为空", ErrInvalidParameter)
	}

	request := sms.NewSendSmsRequest()
	/*
		下发手机号码，采用 E.164 标准，格式为+[国家或地区码][手机号]，单次请求最多支持200个手机号且要求全为境内手机号或全为境外手机号。
		例如：+8618501234444， 其中前面有一个+号 ，86为国家码，18501234444为手机号。
		注：发送国内短信格式还支持0086、86或无任何国家或地区码的11位手机号码，前缀默认为+86。
		示例值：["+8618501234444"]
	*/
	phoneNumPtrs := make([]*string, len(req.PhoneNumbers))
	for i := range req.PhoneNumbers {
		// 如果手机号不是以+开头，则添加+86前缀（中国大陆）
		fullPhoneNum := req.PhoneNumbers[i]
		if !strings.HasPrefix(req.PhoneNumbers[i], "+") {
			fullPhoneNum = "+86" + req.PhoneNumbers[i]
		}
		phoneNumPtr := fullPhoneNum
		phoneNumPtrs[i] = &phoneNumPtr
	}
	request.PhoneNumberSet = phoneNumPtrs

	// 短信 SdkAppId，在 短信控制台 添加应用后生成的实际 SdkAppId，示例如1400006666。
	request.SmsSdkAppId = t.appID
	// 模板 ID，必须填写已审核通过的模板 ID。
	request.TemplateId = &req.TemplateID
	// 短信签名内容，使用 UTF-8 编码，必须填写已审核通过的签名
	request.SignName = &req.SignName

	// 模板参数，若无模板参数，则设置为空。示例值：["4370"]
	var templateParamPtrs []*string
	if req.TemplateParam != nil {
		for _, value := range req.TemplateParam {
			valuePtr := value
			templateParamPtrs = append(templateParamPtrs, &valuePtr)
		}
		request.TemplateParamSet = templateParamPtrs
	}

	response, err := t.client.SendSms(request)
	if err != nil {
		return SendResp{}, fmt.Errorf("%w: %w", ErrSendFailed, err)
	}

	// 确保返回结果中至少有一个发送状态
	if len(response.Response.SendStatusSet) == 0 {
		return SendResp{}, fmt.Errorf("%w: 没有返回发送状态", ErrSendFailed)
	}

	// 构建新的响应格式
	result := SendResp{
		RequestID:    *response.Response.RequestId,
		PhoneNumbers: make(map[string]SendRespStatus),
	}
	for i := range response.Response.SendStatusSet {
		status := response.Response.SendStatusSet[i]
		result.PhoneNumbers[strings.TrimPrefix(*status.PhoneNumber, "+86")] = SendRespStatus{
			Code:    *status.Code,
			Message: *status.Message,
		}
	}
	return result, nil
}
