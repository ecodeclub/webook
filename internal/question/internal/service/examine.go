// Copyright 2023 ecodeclub
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package service

import (
	"context"
	"strings"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/ai"
	"github.com/ecodeclub/webook/internal/question/internal/domain"
	"github.com/ecodeclub/webook/internal/question/internal/repository"
	"github.com/lithammer/shortuuid/v4"
)

var ErrInsufficientCredit = ai.ErrInsufficientCredit

// ExamineService 测试服务
//
//go:generate mockgen -source=./examine.go -destination=../../mocks/examine.mock.go -package=quemocks -typed=true ExamineService
type ExamineService interface {
	// Examine 测试服务
	// input 是用户输入的内容
	Examine(ctx context.Context, uid, qid int64, input string) (domain.ExamineResult, error)
	QuestionResult(ctx context.Context, uid, qid int64) (domain.Result, error)
	GetResults(ctx context.Context, uid int64, ids []int64) (map[int64]domain.ExamineResult, error)
	Correct(ctx context.Context, uid int64, qid int64, questionResult domain.Result) error
}

var _ ExamineService = &LLMExamineService{}

// LLMExamineService 使用 LLM 进行评价的测试服务
type LLMExamineService struct {
	queRepo repository.Repository
	repo    repository.ExamineRepository
	aiSvc   ai.LLMService
}

func (svc *LLMExamineService) GetResults(ctx context.Context, uid int64, ids []int64) (map[int64]domain.ExamineResult, error) {
	results, err := svc.repo.GetResultsByIds(ctx, uid, ids)
	return slice.ToMap[domain.ExamineResult, int64](results, func(ele domain.ExamineResult) int64 {
		return ele.Qid
	}), err
}

func (svc *LLMExamineService) QuestionResult(ctx context.Context, uid, qid int64) (domain.Result, error) {
	return svc.repo.GetResultByUidAndQid(ctx, uid, qid)
}

func (svc *LLMExamineService) Examine(ctx context.Context,
	uid int64,
	qid int64, input string) (domain.ExamineResult, error) {
	const biz = "question_examine"
	que, err := svc.queRepo.GetPubByID(ctx, qid)
	if err != nil {
		return domain.ExamineResult{}, err
	}
	tid := shortuuid.New()
	aiReq := ai.LLMRequest{
		Uid: uid,
		Tid: tid,
		Biz: biz,
		// 标题，标准答案，输入
		Input: []string{que.Title, que.Answer.String(), input},
	}
	aiResp, err := svc.aiSvc.Invoke(ctx, aiReq)
	if err != nil {
		return domain.ExamineResult{}, err
	}
	// 解析结果
	parsedRes := svc.parseExamineResult(aiResp.Answer)
	result := domain.ExamineResult{
		Result:    parsedRes,
		RawResult: aiResp.Answer,
		Tokens:    aiResp.Tokens,
		Amount:    aiResp.Amount,
		Tid:       tid,
	}
	// 开始记录结果
	err = svc.repo.SaveResult(ctx, uid, qid, result)
	return result, err
}

func (svc *LLMExamineService) Correct(ctx context.Context, uid int64,
	qid int64, questionResult domain.Result) error {
	// 更新结果
	return svc.repo.UpdateQuestionResult(ctx, uid, qid, questionResult)
}

func (svc *LLMExamineService) parseExamineResult(answer string) domain.Result {
	answer = strings.TrimSpace(answer)
	// 获取第二行
	segs := strings.SplitN(answer, "\n", 3)
	if len(segs) < 2 {
		return domain.ResultFailed
	}
	// 说明 AI 没有按照我要求的格式返回
	if !strings.Contains(segs[0], "最终评分") {
		return domain.ResultFailed
	}
	// 第一个字符表示的数字
	result := strings.TrimSpace(segs[1])[0] - '0'
	firstZeroIdx := svc.findFirstZeroPosition(result)
	switch firstZeroIdx {
	case 1:
		return domain.ResultBasic
	case 2:
		return domain.ResultIntermediate
	case 3:
		return domain.ResultAdvanced
	default:
		return domain.ResultFailed
	}
}

// findFirstZeroPosition 从右至左找到第一个 0 的位置
func (svc *LLMExamineService) findFirstZeroPosition(b byte) int {
	for i := 0; i < 8; i++ {
		if (b & (1 << i)) == 0 {
			return i
		}
	}
	return -1 // 如果没有找到 0，返回 -1
}

func NewLLMExamineService(
	queRepo repository.Repository,
	repo repository.ExamineRepository,
	aiSvc ai.LLMService,
) ExamineService {
	return &LLMExamineService{
		queRepo: queRepo,
		repo:    repo,
		aiSvc:   aiSvc,
	}
}
