package domain

const (
	BizResumeSkillKeyPoints   = "biz_resume_skill_keypoints"
	BizSkillsRewrite          = "biz_skills_rewrite"
	BizResumeProjectKeyPoints = "biz_resume_project_keypoints"
	BizResumeProjectRewrite   = "biz_resume_project_rewrite"
	BizResumeJobsKeyPoints    = "biz_resume_jobs_keypoints"
	BizResumeJobsRewrite      = "biz_resume_jobs_rewrite"
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
