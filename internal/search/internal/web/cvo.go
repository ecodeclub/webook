package web

import (
	"fmt"
	"strings"

	"github.com/ecodeclub/webook/internal/cases"
	"github.com/ecodeclub/webook/internal/interactive"

	"github.com/ecodeclub/webook/internal/search/internal/domain"
)

type CSearchResp struct {
	Questions   []CSearchRes `json:"questions,omitempty"`
	Cases       []CSearchRes `json:"cases,omitempty"`
	Skills      []CSearchRes `json:"skills,omitempty"`
	QuestionSet []CSearchRes `json:"questionSet,omitempty"`
}

type Interactive struct {
	CollectCnt int  `json:"collectCnt"`
	LikeCnt    int  `json:"likeCnt"`
	ViewCnt    int  `json:"viewCnt"`
	Liked      bool `json:"liked"`
	Collected  bool `json:"collected"`
}
type CSearchRes struct {
	Id          int64       `json:"id,omitempty"`
	Title       string      `json:"title,omitempty"`
	Description string      `json:"description,omitempty"`
	Tags        []string    `json:"tags,omitempty"`
	Result      uint8       `json:"result,omitempty"`
	Interactive Interactive `json:"interactive,omitempty"`
	Utime       int64       `json:"utime,omitempty"`
}

func newInteractive(intr interactive.Interactive) Interactive {
	return Interactive{
		CollectCnt: intr.CollectCnt,
		ViewCnt:    intr.ViewCnt,
		LikeCnt:    intr.LikeCnt,
		Liked:      intr.Liked,
		Collected:  intr.Collected,
	}
}

func newQuestionCSearchRes(que domain.Question, intr interactive.Interactive) CSearchRes {
	res := CSearchRes{
		Id:    que.ID,
		Title: que.Title,
		Tags:  que.Labels,
		Utime: que.Utime.UnixMilli(),
	}
	res.Interactive = newInteractive(intr)
	res.Description = buildQuestionDescription(que)
	return res
}

func newSkillCSearchRes(skill domain.Skill) CSearchRes {
	res := CSearchRes{
		Id:    skill.ID,
		Title: skill.Name,
		Tags:  skill.Labels,
		Utime: skill.Utime.UnixMilli(),
	}
	res.Description = buildSKillDescription(skill)
	return res
}

// truncateString 安全地截取字符串，确保不会截断汉字
func truncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen])
}

// buildQuestionDescription 构建问题的描述信息
func buildQuestionDescription(que domain.Question) string {
	var descBuilder strings.Builder

	// 定义需要处理的描述项
	descItems := []struct {
		prefix string
		val    domain.EsVal
	}{
		{"描述", que.Content},
		{"题目分析", que.Answer.Analysis.Content},
		{"基础回答", que.Answer.Basic.Content},
		{"中级回答", que.Answer.Intermediate.Content},
		{"高级回答", que.Answer.Advanced.Content},
	}

	// 处理每个描述项
	for _, item := range descItems {
		if len(item.val.HighLightVals) > 0 {
			descBuilder.WriteString(fmt.Sprintf("%s：%s<br/>", item.prefix, item.val.HighLightVals[0]))
		}
	}

	// 如果没有找到任何高亮内容，使用原始内容的前100个字符
	if descBuilder.Len() == 0 && que.Content.Val != "" {
		descBuilder.WriteString(truncateString(que.Content.Val, 100))
	}

	return strings.TrimSpace(descBuilder.String())
}

func buildSKillDescription(sk domain.Skill) string {
	var descBuilder strings.Builder
	// 定义需要处理的描述项
	descItems := []struct {
		prefix string
		val    domain.EsVal
	}{
		{"描述", sk.Desc},
		{"基础回答", sk.Basic.Desc},
		{"中级回答", sk.Intermediate.Desc},
		{"高级回答", sk.Advanced.Desc},
	}

	// 处理每个描述项
	for _, item := range descItems {
		if len(item.val.HighLightVals) > 0 {
			descBuilder.WriteString(fmt.Sprintf("%s：%s<br/>", item.prefix, item.val.HighLightVals[0]))
		}
	}

	// 如果没有找到任何高亮内容，使用原始内容的前100个字符
	if descBuilder.Len() == 0 {
		descBuilder.WriteString(truncateString(sk.Desc.Val, 100))
	}
	return strings.TrimSpace(descBuilder.String())
}

func newCaseCSearchRes(ca domain.Case, intr interactive.Interactive) CSearchRes {
	res := CSearchRes{
		Id:    ca.Id,
		Title: ca.Title,
		Tags:  ca.Labels,
		Utime: ca.Utime.UnixMilli(),
	}
	if len(ca.Content.HighLightVals) > 0 {
		res.Description = ca.Content.HighLightVals[0]
	} else {
		res.Description = truncateString(ca.Content.Val, 100)
	}
	res.Interactive = newInteractive(intr)
	return res
}

func newQuestionSetCSearchRes(qs domain.QuestionSet) CSearchRes {
	res := CSearchRes{
		Id:    qs.Id,
		Title: qs.Title,
		Utime: qs.Utime.UnixMilli(),
	}
	if len(qs.Description.HighLightVals) > 0 {
		res.Description = qs.Description.HighLightVals[0]
	} else {
		res.Description = truncateString(qs.Description.Val, 100)
	}
	return res
}

func NewCSearchResult(res *domain.SearchResult,
	examMap map[int64]cases.ExamineResult,
	caseIntr map[int64]interactive.Interactive,
	questionIntr map[int64]interactive.Interactive,
) CSearchResp {
	var newResult CSearchResp
	for _, oldCase := range res.Cases {
		newCase := newCaseCSearchRes(oldCase, caseIntr[oldCase.Id])
		if examMap != nil {
			exam, ok := examMap[oldCase.Id]
			if ok {
				newCase.Result = exam.Result.ToUint8()
			}
		}

		newResult.Cases = append(newResult.Cases, newCase)
	}
	for _, question := range res.Questions {
		newQuestion := newQuestionCSearchRes(question, questionIntr[question.ID])
		newResult.Questions = append(newResult.Questions, newQuestion)
	}
	for _, skill := range res.Skills {
		newResult.Skills = append(newResult.Skills, newSkillCSearchRes(skill))
	}
	for _, questionSet := range res.QuestionSet {
		newResult.QuestionSet = append(newResult.QuestionSet, newQuestionSetCSearchRes(questionSet))
	}

	return newResult
}
