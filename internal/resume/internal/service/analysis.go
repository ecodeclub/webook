package service

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/ecodeclub/webook/internal/ai"
	"github.com/ecodeclub/webook/internal/resume/internal/domain"
	"github.com/lithammer/shortuuid/v4"
	"golang.org/x/sync/errgroup"
)

var ErrInsufficientCredit = ai.ErrInsufficientCredit

type AnalysisService interface {
	Analysis(ctx context.Context, uid int64, resume string) (domain.ResumeAnalysis, error)
}

type analysisService struct {
	aiSvc ai.LLMService
}

func NewAnalysisService(aiSvc ai.LLMService) AnalysisService {
	return &analysisService{
		aiSvc: aiSvc,
	}
}

func (r *analysisService) Analysis(ctx context.Context, uid int64, resume string) (domain.ResumeAnalysis, error) {
	tid := shortuuid.New()
	var eg errgroup.Group

	var amount int64
	var rewriteSKills, rewriteProject, rewriteJobs string
	// 重写技能
	eg.Go(func() error {
		keyPointsAmount, keyPoints, err := r.skillKeypoint(ctx, uid, fmt.Sprintf("%s_skills_get_keypoints", tid), resume)
		if err != nil {
			return err
		}
		atomic.AddInt64(&amount, keyPointsAmount)
		// 暂时不需要传入原始简历，不然会严重超时，并且上下文太长，搞崩系统
		rewriteSkillsAmount, ans, err := r.rewriteSkills(ctx, uid, fmt.Sprintf("%s_skills_rewrite", tid), keyPoints, "")
		if err != nil {
			return err
		}
		atomic.AddInt64(&amount, rewriteSkillsAmount)
		rewriteSKills = ans
		return nil
	})
	// 重写项目
	eg.Go(func() error {
		rewriteProjectAmount, ans, err := r.rewriteProject(ctx, uid, fmt.Sprintf("%s_project_rewrite", tid), resume)
		if err != nil {
			return err
		}
		atomic.AddInt64(&amount, rewriteProjectAmount)
		evaluationAmt, evaluation, err := r.evaluatePrj(ctx, uid,
			domain.BizResumeProjectEvaluation,
			fmt.Sprintf("%s_project_get_evaludation", tid), ans)
		if err != nil {
			return err
		}
		atomic.AddInt64(&amount, evaluationAmt)
		rewriteProject = fmt.Sprintf("%s\n## 综合评价\n%s", ans, evaluation)
		return nil
	})
	// 重写工作经历
	eg.Go(func() error {
		// 暂时还不需要提取关键字
		//keyPointsAmount, keyPoints, err := r.evaluatePrj(ctx, uid, domain.BizResumeJobsKeyPoints, fmt.Sprintf("%s_jobs_get_keypoints", tid), resume)
		//if err != nil {
		//	return err
		//}
		//atomic.AddInt64(&amount, keyPointsAmount)
		rewriteJobsAmount, ans, err := r.rewriteJobs(ctx, uid, fmt.Sprintf("%s_jobs_rewrite", tid), "", resume)
		if err != nil {
			return err
		}
		atomic.AddInt64(&amount, rewriteJobsAmount)
		rewriteJobs = ans
		return nil
	})

	if err := eg.Wait(); err != nil {
		return domain.ResumeAnalysis{}, err
	}

	return domain.ResumeAnalysis{
		Amount:         amount,
		RewriteSkills:  rewriteSKills,
		RewriteProject: rewriteProject,
		RewriteJobs:    rewriteJobs,
	}, nil

}

func (r *analysisService) evaluatePrj(ctx context.Context, uid int64, biz, tid, rewritePrj string) (int64, string, error) {
	aiReq := ai.LLMRequest{
		Uid: uid,
		Tid: tid,
		Biz: biz,
		// 标题，标准答案，输入
		Input: []string{rewritePrj},
	}
	resp, err := r.aiSvc.Invoke(ctx, aiReq)
	if err != nil {
		return 0, "", err
	}
	return resp.Amount, resp.Answer, nil
}

// 提取关键字
func (r *analysisService) skillKeypoint(ctx context.Context, uid int64, tid, resume string) (int64, string, error) {
	aiReq := ai.LLMRequest{
		Uid:   uid,
		Tid:   tid,
		Biz:   domain.BizResumeSkillKeyPoints,
		Input: []string{resume},
	}
	resp, err := r.aiSvc.Invoke(ctx, aiReq)
	if err != nil {
		return 0, "", err
	}
	return resp.Amount, resp.Answer, nil
}

// 重写技能
func (r *analysisService) rewriteSkills(ctx context.Context, uid int64, tid, keyPoints, resume string) (int64, string, error) {
	aiReq := ai.LLMRequest{
		Uid: uid,
		Tid: tid,
		Biz: domain.BizSkillsRewrite,
		// 标题，标准答案，输入
		Input: []string{
			// 简历
			resume,
			// 前一步提取的关键字
			keyPoints,
		},
	}
	resp, err := r.aiSvc.Invoke(ctx, aiReq)
	if err != nil {
		return 0, "", err
	}
	return resp.Amount, resp.Answer, nil
}

// 重写项目
func (r *analysisService) rewriteProject(ctx context.Context, uid int64, tid, resume string) (int64, string, error) {
	aiReq := ai.LLMRequest{
		Uid: uid,
		Tid: tid,
		Biz: domain.BizResumeProjectRewrite,
		// 标题，标准答案，输入
		Input: []string{
			resume,
		},
	}
	resp, err := r.aiSvc.Invoke(ctx, aiReq)
	if err != nil {
		return 0, "", err
	}
	return resp.Amount, resp.Answer, nil
}

// 重写工作经历
func (r *analysisService) rewriteJobs(ctx context.Context, uid int64, tid, keyPoints, resume string) (int64, string, error) {
	aiReq := ai.LLMRequest{
		Uid: uid,
		Tid: tid,
		Biz: domain.BizResumeJobsRewrite,
		// 标题，标准答案，输入
		Input: []string{
			resume,
			keyPoints,
		},
	}
	resp, err := r.aiSvc.Invoke(ctx, aiReq)
	if err != nil {
		return 0, "", err
	}
	return resp.Amount, resp.Answer, nil
}
