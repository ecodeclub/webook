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

// ExamineService 测试服务
type ExamineService interface {
	// Examine 测试服务
	// input 是用户输入的内容
	Examine(ctx context.Context, uid, qid int64, input string) (domain.ExamineResult, error)
	QuestionResult(ctx context.Context, uid, qid int64) (domain.Result, error)
	GetResults(ctx context.Context, uid int64, ids []int64) (map[int64]domain.ExamineResult, error)
}

var _ ExamineService = &GPTExamineService{}

// GPTExamineService 使用 GPT 进行评价的测试服务
type GPTExamineService struct {
	queRepo repository.Repository
	repo    repository.ExamineRepository
	aiSvc   ai.GPTService
}

func (svc *GPTExamineService) GetResults(ctx context.Context, uid int64, ids []int64) (map[int64]domain.ExamineResult, error) {
	results, err := svc.repo.GetResultsByIds(ctx, uid, ids)
	return slice.ToMap[domain.ExamineResult, int64](results, func(ele domain.ExamineResult) int64 {
		return ele.Qid
	}), err
}

func (svc *GPTExamineService) QuestionResult(ctx context.Context, uid, qid int64) (domain.Result, error) {
	return svc.repo.GetResultByUidAndQid(ctx, uid, qid)
}

func (svc *GPTExamineService) Examine(ctx context.Context,
	uid int64,
	qid int64, input string) (domain.ExamineResult, error) {
	// 实际上我们只需要 title，但是懒得写一个新的接口了
	que, err := svc.queRepo.GetPubByID(ctx, qid)
	if err != nil {
		return domain.ExamineResult{}, err
	}
	tid := shortuuid.New()
	aiReq := ai.GPTRequest{
		Uid:   uid,
		Tid:   tid,
		Input: []string{que.Title, input},
	}
	aiResp, err := svc.aiSvc.Invoke(ctx, aiReq)
	if err != nil {
		return domain.ExamineResult{}, nil
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

func (svc *GPTExamineService) parseExamineResult(answer string) domain.Result {
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

func NewGPTExamineService(
	queRepo repository.Repository,
	repo repository.ExamineRepository,
) ExamineService {
	return &GPTExamineService{
		queRepo: queRepo,
		repo:    repo,
		aiSvc:   &AiService{},
	}
}
