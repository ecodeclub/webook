package email

import "context"

//go:generate mockgen -source=./type.go -package=emailmocks -destination=./mocks/email.mock.go -typed Service
type Service interface {
	SendMail(ctx context.Context, mail Mail) error
}

type Mail struct {
	From        string
	To          string
	Subject     string
	Body        []byte
	Attachments []Attachment
}

type Attachment struct {
	Filename string
	Content  []byte // 文件内容（部分渠道不支持直传，需要先上传，见 URL）
	URL      string // 可公开访问的 HTTP(S) 地址；Aliyun DirectMail 需要该字段
}
