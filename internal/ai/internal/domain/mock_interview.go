package domain

import (
	"errors"
	"strings"
)

// MockInterview 表示一场模拟面试
type MockInterview struct {
	ID         int64
	ChatSN     string
	Uid        int64
	Title      string
	Evaluation map[string]any
	Ctime      int64
	Utime      int64
}

// MockInterviewQuestion 表示某场模拟面试中的一道题
type MockInterviewQuestion struct {
	ID          int64
	InterviewID int64
	ChatSN      string
	Uid         int64
	Biz         string
	BizID       int64
	Title       string
	Answer      map[string]any
	Evaluation  map[string]any
	Ctime       int64
	Utime       int64
}

// Validate 校验题目来源规则
func (q MockInterviewQuestion) Validate() error {
	if q.Biz == "" {
		return errors.New("面试题目非法: biz不能为空")
	}
	if strings.HasPrefix(q.Biz, "generated") {
		if q.Title == "" {
			return errors.New("面试题目非法: 生成题目“标题”不能为空")
		}
	} else {
		if q.BizID == 0 || q.Title != "" {
			return errors.New("面试题目非法: 现有题目必须设置biz_id且不能设置标题")
		}
	}
	return nil
}
