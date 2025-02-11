package domain

const ReviewBiz = "review"

type Review struct {
	ID               int64
	Title            string
	Desc             string
	Labels           []string
	Uid              int64
	JD               string
	JDAnalysis       string
	Questions        string
	QuestionAnalysis string
	Resume           string
	Status           ReviewStatus
	Utime            int64
}
type ReviewStatus uint8

func (s ReviewStatus) ToUint8() uint8 {
	return uint8(s)
}

const (
	// UnknownStatus 未知
	UnknownStatus ReviewStatus = 0
	// UnPublishedStatus 未发布
	UnPublishedStatus ReviewStatus = 1
	// PublishedStatus 发布
	PublishedStatus ReviewStatus = 2
)
