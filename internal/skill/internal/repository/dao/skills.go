package dao

import (
	"context"
	"time"

	"github.com/ego-component/egorm"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SkillDAO interface {
	// Create 管理端接口
	Create(ctx context.Context, skill Skill, skillLevels []SkillLevel) (int64, error)
	// Update 管理端接口
	Update(ctx context.Context, skill Skill, skillLevels []SkillLevel) error
	SaveRefs(ctx context.Context, reqs []SkillRef) error
	// List 列表
	List(ctx context.Context, offset, limit int) ([]Skill, error)
	// Info 详情
	Info(ctx context.Context, id int64) (Skill, error)
	SkillLevelInfo(tx context.Context, id int64) ([]SkillLevel, error)
	// Refs id 为skill的id
	Refs(ctx context.Context, id int64) ([]SkillRef, error)
	Count(ctx context.Context) (int64, error)
}

type skillDAO struct {
	db *egorm.Component
}

func (s *skillDAO) Create(ctx context.Context, skill Skill, skillLevels []SkillLevel) (int64, error) {
	var id int64
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		skill, _, err = s.create(tx, skill, skillLevels)
		id = skill.Id
		return err
	})
	return id, err
}

func (s *skillDAO) create(tx *gorm.DB, skill Skill, skillLevels []SkillLevel) (Skill, []SkillLevel, error) {
	skill.Utime = time.Now().UnixMilli()
	skill.Ctime = time.Now().UnixMilli()
	err := tx.Clauses(clause.OnConflict{
		DoUpdates: clause.AssignmentColumns([]string{
			"labels", "desc", "utime",
		}),
		Columns: []clause.Column{{Name: "name"}},
	}).Create(&skill).Error
	if err != nil {
		return skill, nil, err
	}
	for i := range skillLevels {
		skillLevels[i].Utime = time.Now().UnixMilli()
		skillLevels[i].Ctime = time.Now().UnixMilli()
		skillLevels[i].Sid = skill.Id
	}
	err = tx.Clauses(clause.OnConflict{
		DoUpdates: clause.AssignmentColumns([]string{
			"desc", "utime",
		}),
		Columns: []clause.Column{{Name: "sid"}, {Name: "level"}},
	}).Create(&skillLevels).Error
	if err != nil {
		return skill, nil, err
	}
	return skill, skillLevels, nil
}

func (s *skillDAO) Update(ctx context.Context, skill Skill, skillLevels []SkillLevel) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		_, _, err := s.update(tx, skill, skillLevels)
		return err
	})
}

func (s *skillDAO) update(tx *gorm.DB, skill Skill, skillLevels []SkillLevel) (Skill, []SkillLevel, error) {
	err := tx.Model(&skill).Where("id = ?", skill.Id).Updates(map[string]any{
		"labels": skill.Labels,
		"name":   skill.Name,
		"desc":   skill.Desc,
		"utime":  time.Now().UnixMilli(),
	}).Error
	if err != nil {
		return skill, skillLevels, err
	}
	for i := range skillLevels {
		skillLevels[i].Sid = skill.Id
		skillLevels[i].Utime = time.Now().UnixMilli()
		skillLevels[i].Ctime = time.Now().UnixMilli()
	}
	err = tx.Clauses(clause.OnConflict{
		DoUpdates: clause.AssignmentColumns([]string{
			"desc", "utime", "level",
		}),
	}).Create(&skillLevels).Error
	return skill, skillLevels, err
}

func (s *skillDAO) SaveRefs(ctx context.Context, refs []SkillRef) error {
	if len(refs) == 0 {
		return nil
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Where("sid = ?", refs[0].Sid).Delete(&SkillRef{}).Error
		if err != nil {
			return err
		}
		return tx.Create(&refs).Error
	})
}

func (s *skillDAO) List(ctx context.Context, offset, limit int) ([]Skill, error) {
	var skills []Skill
	err := s.db.WithContext(ctx).Model(&Skill{}).
		Order("id desc").
		Offset(offset).Limit(limit).Find(&skills).Error
	return skills, err
}

func (s *skillDAO) Info(ctx context.Context, id int64) (Skill, error) {
	var skill Skill
	err := s.db.WithContext(ctx).Model(&Skill{}).Where("id = ? ", id).First(&skill).Error
	return skill, err
}

func (s *skillDAO) SkillLevelInfo(ctx context.Context, id int64) ([]SkillLevel, error) {
	var skillLevels []SkillLevel
	err := s.db.WithContext(ctx).Model(&SkillLevel{}).Where("sid = ? ", id).Find(&skillLevels).Error
	return skillLevels, err
}

func (s *skillDAO) Refs(ctx context.Context, id int64) ([]SkillRef, error) {
	var reqs []SkillRef
	err := s.db.WithContext(ctx).Model(&SkillRef{}).
		Where("sid = ?", id).Find(&reqs).Error
	return reqs, err
}

func (s *skillDAO) Count(ctx context.Context) (int64, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&Skill{}).Count(&count).Error
	return count, err
}

func NewSkillDAO(db *egorm.Component) SkillDAO {
	return &skillDAO{
		db: db,
	}
}
