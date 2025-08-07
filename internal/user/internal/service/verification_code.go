package service

import (
	"context"
	"crypto/rand"
	"fmt"

	"github.com/ecodeclub/webook/internal/sms/client"
	"github.com/ecodeclub/webook/internal/user/internal/repository"
	"github.com/pkg/errors"
)

const templateID = "SMS_315530322"

var (
	ErrVerificationCode = errors.New("验证码错误")
)

type VerificationCodeSvc interface {
	Send(ctx context.Context, phone string) error
	Verify(ctx context.Context, phone string, code string) error
}

type smsServiceImpl struct {
	client client.Client
	repo   repository.VerificationCodeRepo
}

func NewVerificationCodeSvc(client client.Client,
	repo repository.VerificationCodeRepo,
) VerificationCodeSvc {
	return &smsServiceImpl{
		client: client,
		repo:   repo,
	}
}

func (s *smsServiceImpl) Verify(ctx context.Context, phone string, code string) error {
	wantCode, err := s.repo.GetPhoneCode(ctx, phone)
	if err != nil {
		return err
	}
	if wantCode != code {
		return ErrVerificationCode
	}
	return nil
}

func (s *smsServiceImpl) GetCode(ctx context.Context, phone string) (string, error) {
	return s.repo.GetPhoneCode(ctx, phone)
}

func (s *smsServiceImpl) Send(ctx context.Context, phone string) error {
	code := s.generateCode()
	err := s.repo.SetPhoneCode(ctx, phone, code)
	if err != nil {
		return err
	}
	// todo 到时候审核通过需要修改
	params := map[string]string{
		"code": code,
	}
	respMap, err := s.client.Send(client.SendReq{
		PhoneNumbers:  []string{phone},
		TemplateID:    templateID,
		TemplateParam: params,
	})
	if err != nil {
		return err
	}
	resp := respMap.PhoneNumbers[phone]
	if resp.Code != client.OK {
		return errors.New(resp.Message)
	}
	return nil
}

func (s *smsServiceImpl) generateCode() string {
	// 使用crypto/rand生成随机字节
	bytes := make([]byte, 6)
	_, _ = rand.Read(bytes)
	// 将字节转换为六位数字验证码
	code := ""
	for _, b := range bytes {
		// 将字节值映射到0-9范围
		digit := int(b) % 10
		code += fmt.Sprintf("%d", digit)
	}
	return code
}
