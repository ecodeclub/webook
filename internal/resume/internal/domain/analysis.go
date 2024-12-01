package domain

const (
	BizResumeSkillKeyPoints = "biz_resume_skill_keypoints"
	BizSkillsRewrite        = "biz_resume_skill_rewrite"
	// BizResumeProjectEvaluation 评价项目经历
	BizResumeProjectEvaluation = "biz_resume_project_evaluation"
	BizResumeProjectRewrite    = "biz_resume_project_rewrite"
	//BizResumeJobsKeyPoints    = "biz_resume_job_keypoints"
	BizResumeJobsRewrite = "biz_resume_job_rewrite"
)

type ResumeAnalysis struct {
	Amount int64
	// 技能
	RewriteSkills string
	// 项目
	RewriteProject string
	// 工作经历
	RewriteJobs string
}
