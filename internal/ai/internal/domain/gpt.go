package domain

const BizQuestionExamine = "question_examine"

type GPTRequest struct {
	Biz string
	Uid int64
	// 请求id
	Tid string
	// 用户的输入
	Input []string
	// Prompt 将 input 和 PromptTemplate 结合之后生成的正儿八经的 Prompt
	Prompt string
	// 业务相关的配置
	Config BizConfig
}

type GPTResponse struct {
	// 花费的token
	Tokens int64
	// 花费的金额
	Amount int64
	// gpt的回答
	Answer string
}

type BizConfig struct {
	// 允许的最长输入
	// 这里我们不用计算 token，只需要简单约束一下字符串长度就可以
	MaxInput int
	// 使用的知识库
	KnowledgeId string
	// 提示词。虽然这里只有一个 PromptTemplate 字段，
	// 但是在部分业务里面，它是一个 json
	// 这里一般使用 %s
	// 后续考虑 key value 的形式
	PromptTemplate string
}

type GPTCredit struct {
	Id     int64
	Tid    string
	Uid    int64
	Biz    string
	Tokens int64
	Amount int64
	Status CreditStatus
	Ctime  int64
	Utime  int64
}

type GPTRecord struct {
	Id             int64
	Tid            string
	Uid            int64
	Biz            string
	Tokens         int64
	Amount         int64
	Input          []string
	Status         RecordStatus
	KnowledgeId    string
	PromptTemplate string
	Answer         string
	Ctime          int64
	Utime          int64
}

type CreditStatus uint8

const (
	CreditStatusProcessing CreditStatus = iota
	CreditStatusSuccess
	CreditStatusFailed
)

func (g CreditStatus) ToUint8() uint8 {
	return uint8(g)
}

type RecordStatus uint8

func (g RecordStatus) ToUint8() uint8 {
	return uint8(g)
}

const (
	RecordStatusProcessing RecordStatus = 0
	RecordStatusSuccess    RecordStatus = 1
	RecordStatusFailed     RecordStatus = 2
)
