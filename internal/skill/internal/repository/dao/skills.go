package dao

import (
	"context"
	"time"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ego-component/egorm"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SkillDAO interface {
	// 管理端接口
	// Create 和 Update 返回值为  skillLevel的id
	Create(ctx context.Context, skill Skill, skillLevels []SkillLevel) (int64, error)
	Update(ctx context.Context, skill Skill, skillLevels []SkillLevel) error
	UpdateRequest(ctx context.Context, sid, slid int64, reqs []SkillPreRequest) error
	// skill同步
	SyncSkill(ctx context.Context, skill Skill, skillLevel []SkillLevel) (int64, error)
	// id 为skillLevel的id
	SyncSKillRequest(ctx context.Context, sid, slid int64, reqs []SkillPreRequest) error
	// 列表
	List(ctx context.Context, offset, limit int) ([]Skill, error)
	// 详情
	Info(ctx context.Context, id int64) (Skill, error)
	SkillLevelInfo(tx context.Context, id int64) ([]SkillLevel, error)
	// id 为skill的id
	RequestInfo(ctx context.Context, id int64) ([]SkillPreRequest, error)
	Count(ctx context.Context) (int64, error)
	// c端
	Publist(ctx context.Context, offset int, limit int) ([]PubSkill, error)
	PubCount(ctx context.Context) (int64, error)
	PubInfo(ctx context.Context, id int64) (PubSkill, error)
	PubLevels(ctx context.Context, id int64) ([]PubSkillLevel, error)
	PubRequestInfo(ctx context.Context, id int64) ([]PubSKillPreRequest, error)
}

type skillDAO struct {
	db *egorm.Component
}

func NewSkillDAO(db *egorm.Component) SkillDAO {
	return &skillDAO{
		db: db,
	}
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

func (s *skillDAO) UpdateRequest(ctx context.Context, sid, slid int64, reqs []SkillPreRequest) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.updateRequest(tx, sid, slid, reqs)
	})
}

func (s *skillDAO) updateRequest(tx *gorm.DB, sid, slid int64, reqs []SkillPreRequest) error {
	// 删除所有案例
	err := tx.Model(&SkillPreRequest{}).Where("sid =? and slid = ?", sid, slid).Delete(&reqs).Error
	if err != nil {
		return err
	}
	for i := range reqs {
		reqs[i].Ctime = time.Now().UnixMilli()
		reqs[i].Utime = time.Now().UnixMilli()
	}
	return tx.Model(&SkillPreRequest{}).Create(reqs).Error
}

func (s *skillDAO) SyncSkill(ctx context.Context, skill Skill, skillLevels []SkillLevel) (int64, error) {
	id := skill.Id
	skill.Ctime = time.Now().UnixMilli()
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		if skill.Id == 0 {
			skill, skillLevels, err = s.create(tx, skill, skillLevels)
			id = skill.Id
			if err != nil {
				return err
			}
		} else {
			skill, skillLevels, err = s.update(tx, skill, skillLevels)
			if err != nil {
				return err
			}
		}
		pubSkillLevels := slice.Map(skillLevels, func(idx int, src SkillLevel) PubSkillLevel {
			return PubSkillLevel(src)
		})
		return s.savePubSkill(tx, PubSkill(skill), pubSkillLevels)
	})
	return id, err

}

func (s *skillDAO) savePubSkill(tx *gorm.DB, skill PubSkill, skillLevels []PubSkillLevel) error {
	err := tx.Save(&skill).Error
	if err != nil {
		return err
	}
	return tx.Clauses(clause.OnConflict{
		DoUpdates: clause.AssignmentColumns([]string{
			"desc", "utime", "level",
		}),
	}).Create(&skillLevels).Error
}

func (s *skillDAO) SyncSKillRequest(ctx context.Context, sid, slid int64, reqs []SkillPreRequest) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := s.updateRequest(tx, sid, slid, reqs)
		if err != nil {
			return err
		}
		pubSkillPreRequests := slice.Map(reqs, func(idx int, src SkillPreRequest) PubSKillPreRequest {
			return PubSKillPreRequest(src)
		})
		return s.updatePubRequest(tx, sid, slid, pubSkillPreRequests)
	})
}

func (s *skillDAO) updatePubRequest(tx *gorm.DB, sid, slid int64, reqs []PubSKillPreRequest) error {
	err := tx.Model(&PubSKillPreRequest{}).Where("slid = ? and sid = ? ", slid, sid).Delete(&[]PubSKillPreRequest{}).Error
	if err != nil {
		return err
	}
	for i := range reqs {
		reqs[i].Ctime = time.Now().UnixMilli()
		reqs[i].Utime = time.Now().UnixMilli()
	}
	return tx.Model(&PubSKillPreRequest{}).Create(reqs).Error
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

func (s *skillDAO) RequestInfo(ctx context.Context, id int64) ([]SkillPreRequest, error) {
	var reqs []SkillPreRequest
	err := s.db.WithContext(ctx).Model(&SkillPreRequest{}).Where("sid = ?", id).Find(&reqs).Error
	return reqs, err
}

func (s *skillDAO) Count(ctx context.Context) (int64, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&Skill{}).Count(&count).Error
	return count, err
}
