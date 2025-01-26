package dao

import (
	"context"
	"errors"
	"time"

	"github.com/ego-component/egorm"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ResumeProjectDAO interface {
	Upsert(ctx context.Context, pro ResumeProject) (int64, error)
	Delete(ctx context.Context, uid, id int64) error
	Find(ctx context.Context, uid int64) ([]ResumeProject, error)
	First(ctx context.Context, id int64) (ResumeProject, error)
	SaveContribution(ctx context.Context, contribution Contribution, cases []RefCase) (int64, error)
	FindContributions(ctx context.Context, projectId int64) ([]Contribution, error)
	BatchFindContributions(ctx context.Context, projectIds []int64) (map[int64][]Contribution, error)
	FindRefCases(ctx context.Context, contributionIds []int64) (map[int64][]RefCase, error)
	SaveDifficulty(ctx context.Context, difficulty Difficulty) error
	BatchFindDifficulty(ctx context.Context, projectIds []int64) (map[int64][]Difficulty, error)
	FindDifficulties(ctx context.Context, projectId int64) ([]Difficulty, error)
	DeleteDifficulty(ctx context.Context, id int64) error
	DeleteContribution(ctx context.Context, id int64) error
}

type resumeProjectDAO struct {
	db *egorm.Component
}

func (r *resumeProjectDAO) BatchFindContributions(ctx context.Context, projectIds []int64) (map[int64][]Contribution, error) {
	var contributions []Contribution
	err := r.db.WithContext(ctx).Where("project_id in ?", projectIds).Find(&contributions).Error
	if err != nil {
		return nil, err
	}
	contributionMap := make(map[int64][]Contribution, len(projectIds))
	for _, contribution := range contributions {
		cs, ok := contributionMap[contribution.ProjectID]
		if ok {
			cs = append(cs, contribution)
			contributionMap[contribution.ProjectID] = cs
		} else {
			contributionMap[contribution.ProjectID] = []Contribution{
				contribution,
			}
		}
	}
	return contributionMap, nil
}

func (r *resumeProjectDAO) BatchFindDifficulty(ctx context.Context, projectIds []int64) (map[int64][]Difficulty, error) {
	var difficulties []Difficulty
	err := r.db.WithContext(ctx).Where("project_id in ?", projectIds).Find(&difficulties).Error
	if err != nil {
		return nil, err
	}
	diffMap := make(map[int64][]Difficulty, len(projectIds))
	for _, diff := range difficulties {
		diffs, ok := diffMap[diff.ProjectID]
		if ok {
			diffs = append(diffs, diff)
			diffMap[diff.ProjectID] = diffs
		} else {
			diffMap[diff.ProjectID] = []Difficulty{
				diff,
			}
		}
	}
	return diffMap, nil
}

func NewResumeProjectDAO(db *egorm.Component) ResumeProjectDAO {
	return &resumeProjectDAO{
		db: db,
	}
}

func (r *resumeProjectDAO) DeleteContribution(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.WithContext(ctx).Where("id = ?", id).Delete(&Contribution{}).Error
		if err != nil {
			return err
		}
		err = tx.WithContext(ctx).Model(&RefCase{}).Where("contribution_id = ?", id).Delete(&RefCase{}).Error
		return err
	})
}

func (r *resumeProjectDAO) DeleteDifficulty(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Where("id = ? ", id).Delete(&Difficulty{}).Error
}

func (r *resumeProjectDAO) FindDifficulties(ctx context.Context, projectId int64) ([]Difficulty, error) {
	var difficulties []Difficulty
	err := r.db.WithContext(ctx).Where("project_id=?", projectId).Find(&difficulties).Error
	return difficulties, err
}

func (r *resumeProjectDAO) SaveDifficulty(ctx context.Context, difficulty Difficulty) error {
	now := time.Now().UnixMilli()
	difficulty.Utime = now
	difficulty.Ctime = now
	err := r.db.WithContext(ctx).Model(&Difficulty{}).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{"desc", "utime", "case_id", "level"}),
		}).Create(&difficulty).Error
	return err
}

func (r *resumeProjectDAO) FindRefCases(ctx context.Context, contributionIds []int64) (map[int64][]RefCase, error) {
	var refCases []RefCase
	err := r.db.WithContext(ctx).
		Where("contribution_id in ?", contributionIds).
		Find(&refCases).Error
	if err != nil {
		return nil, err
	}
	refCaseMap := make(map[int64][]RefCase, len(contributionIds))
	for _, refCase := range refCases {
		v, ok := refCaseMap[refCase.ContributionID]
		if !ok {
			refCaseMap[refCase.ContributionID] = []RefCase{refCase}
		} else {
			v = append(v, refCase)
			refCaseMap[refCase.ContributionID] = v
		}
	}
	return refCaseMap, err
}

func (r *resumeProjectDAO) FindContributions(ctx context.Context, projectId int64) ([]Contribution, error) {
	var contributions []Contribution
	err := r.db.WithContext(ctx).Where("project_id=?", projectId).Find(&contributions).Error
	return contributions, err
}

func (r *resumeProjectDAO) SaveContribution(ctx context.Context, contribution Contribution, cases []RefCase) (int64, error) {
	var contributionId int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		contributionId, err = r.saveContribution(ctx, tx, contribution)
		if err != nil {
			return err
		}
		for idx := range cases {
			cases[idx].ContributionID = contributionId
		}
		return r.saveContributionCases(ctx, tx, contribution, cases)
	})
	return contributionId, err
}

func (r *resumeProjectDAO) saveContribution(ctx context.Context, tx *gorm.DB, contribution Contribution) (int64, error) {
	now := time.Now().UnixMilli()
	contribution.Ctime = now
	contribution.Utime = now
	err := tx.WithContext(ctx).Model(&Contribution{}).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{"desc", "utime"}),
		}).Create(&contribution).Error
	return contribution.ID, err
}

func (r *resumeProjectDAO) saveContributionCases(ctx context.Context, tx *gorm.DB, contribution Contribution, cases []RefCase) error {
	// 删除所有关联case

	err := tx.WithContext(ctx).
		Where("contribution_id=?", contribution.ID).Delete(&RefCase{}).Error
	if err != nil {
		return err
	}
	if len(cases) == 0 {
		return nil
	}
	now := time.Now().UnixMilli()
	for idx := range cases {
		cases[idx].Ctime = now
		cases[idx].Utime = now
	}
	return tx.WithContext(ctx).Model(&RefCase{}).Create(&cases).Error
}

func (r *resumeProjectDAO) Upsert(ctx context.Context, pro ResumeProject) (int64, error) {
	now := time.Now().UnixMilli()
	pro.Utime = now
	pro.Ctime = now
	err := r.db.WithContext(ctx).Model(&ResumeProject{}).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{"start_time", "utime", "end_time", "name", "introduction", "core"}),
		}).Create(&pro).Error
	return pro.ID, err
}

func (r *resumeProjectDAO) Delete(ctx context.Context, uid, id int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&ResumeProject{}).Where("id = ? and uid = ?", id, uid).Delete(&ResumeProject{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected <= 0 {
			return errors.New("删除失败")
		}
		var ids []int64
		err := tx.Model(&Contribution{}).Select("id").Where("project_id = ?", id).Find(&ids).Error
		if err != nil {
			return err
		}
		err = tx.Model(&Contribution{}).Where("project_id = ?", id).Delete(&Contribution{}).Error
		if err != nil {
			return err
		}
		err = tx.Model(&Difficulty{}).Where("project_id = ?", id).Delete(&Difficulty{}).Error
		if err != nil {
			return err
		}
		return tx.Model(&RefCase{}).Where("contribution_id in ?", ids).Delete(&RefCase{}).Error
	})
}

func (r *resumeProjectDAO) Find(ctx context.Context, uid int64) ([]ResumeProject, error) {
	var projects []ResumeProject
	err := r.db.WithContext(ctx).
		Where("uid = ?", uid).
		Order("id desc").
		Find(&projects).Error
	return projects, err
}

func (r *resumeProjectDAO) First(ctx context.Context, id int64) (ResumeProject, error) {
	var p ResumeProject
	err := r.db.WithContext(ctx).Where("id = ? ", id).First(&p).Error
	return p, err
}
