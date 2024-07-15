package domain

type GPTRequest struct {
	Biz string
	Uid int64
	// 请求id
	Tid string
	// 用户的输入
	Input     []string
	BizConfig GPTBiz
}

type GPTResponse struct {
	// 花费的token
	Tokens int
	// 花费的金额
	Amount int64
	// gpt的回答
	Answer string
}

type GPTBiz struct {
	// 业务名称
	Biz string
	// 每个token的钱 分为单位
	AmountPerToken float64
	// 每个token的积分
	CreditPerToken float64
	// 一次最多返回多少Tokens
	MaxTokensPerTime int
}

type GPTCreditLog struct {
	Id     int64
	Tid    string
	Uid    int64
	Biz    string
	Tokens int64
	Amount int64
	Credit int64
	Status GPTLogStatus
	Prompt string
	Answer string
	Ctime  int64
	Utime  int64
}

type GPTLog struct {
	Id     int64
	Tid    string
	Uid    int64
	Biz    string
	Tokens int64
	Amount int64
	Status GPTLogStatus
	Prompt string
	Answer string
	Ctime  int64
	Utime  int64
}

type GPTLogStatus uint8

func (g GPTLogStatus) ToUint8() uint8 {
	return uint8(g)
}

const (
	ProcessingStatus GPTLogStatus = 0
	SuccessStatus    GPTLogStatus = 1
	FailLogStatus    GPTLogStatus = 2
)
