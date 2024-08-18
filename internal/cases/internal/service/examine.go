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
	"github.com/ecodeclub/webook/internal/cases/internal/domain"
	"github.com/ecodeclub/webook/internal/cases/internal/repository"
	"github.com/lithammer/shortuuid/v4"
)

var ErrInsufficientCredit = ai.ErrInsufficientCredit

// ExamineService 测试服务
//
//go:generate mockgen -source=./examine.go -destination=../../mocks/examine.mock.go -package=casemocks  -typed=true ExamineService
type ExamineService interface {
	// Examine 测试服务
	// input 是用户输入的内容
	Examine(ctx context.Context, uid, cid int64, input string) (domain.ExamineCaseResult, error)
	QuestionResult(ctx context.Context, uid, cid int64) (domain.CaseResult, error)
	GetResults(ctx context.Context, uid int64, ids []int64) (map[int64]domain.ExamineCaseResult, error)
}

var _ ExamineService = &LLMExamineService{}

// LLMExamineService 使用 LLM 进行评价的测试服务
type LLMExamineService struct {
	caseRepo repository.CaseRepo
	repo     repository.ExamineRepository
	aiSvc    ai.LLMService
}

func (svc *LLMExamineService) GetResults(ctx context.Context, uid int64, ids []int64) (map[int64]domain.ExamineCaseResult, error) {
	results, err := svc.repo.GetResultsByIds(ctx, uid, ids)
	return slice.ToMap[domain.ExamineCaseResult, int64](results, func(ele domain.ExamineCaseResult) int64 {
		return ele.Cid
	}), err
}

func (svc *LLMExamineService) QuestionResult(ctx context.Context, uid, qid int64) (domain.CaseResult, error) {
	return svc.repo.GetResultByUidAndQid(ctx, uid, qid)
}

func (svc *LLMExamineService) Examine(ctx context.Context,
	uid int64,
	cid int64, input string) (domain.ExamineCaseResult, error) {
	const biz = "case_examine"
	// 实际上我们只需要 title，但是懒得写一个新的接口了
	ca, err := svc.caseRepo.GetPubByID(ctx, cid)
	if err != nil {
		return domain.ExamineCaseResult{}, err
	}
	tid := shortuuid.New()
	aiReq := ai.LLMRequest{
		Uid:   uid,
		Tid:   tid,
		Biz:   biz,
		Input: []string{ca.Title, input},
	}
	aiResp, err := svc.aiSvc.Invoke(ctx, aiReq)
	if err != nil {
		return domain.ExamineCaseResult{}, err
	}
	// 解析结果
	parsedRes := svc.parseExamineResult(aiResp.Answer)
	result := domain.ExamineCaseResult{
		Result:    parsedRes,
		RawResult: aiResp.Answer,
		Tokens:    aiResp.Tokens,
		Amount:    aiResp.Amount,
		Tid:       tid,
	}
	// 开始记录结果
	err = svc.repo.SaveResult(ctx, uid, cid, result)
	return result, err
}

func (svc *LLMExamineService) parseExamineResult(answer string) domain.CaseResult {
	answer = strings.TrimSpace(answer)
	// 获取第一行
	segs := strings.SplitN(answer, "\n", 2)
	if len(segs) < 1 {
		return domain.ResultFailed
	}
	result := segs[0]
	switch {
	case strings.Contains(result, "15K"):
		return domain.ResultBasic
	case strings.Contains(result, "25K"):
		return domain.ResultIntermediate
	case strings.Contains(result, "35K"):
		return domain.ResultAdvanced
	default:
		return domain.ResultFailed
	}
}

func NewLLMExamineService(
	caseRepo repository.CaseRepo,
	repo repository.ExamineRepository,
	aiSvc ai.LLMService,
) ExamineService {
	return &LLMExamineService{
		caseRepo: caseRepo,
		repo:     repo,
		aiSvc:    aiSvc,
	}
}
