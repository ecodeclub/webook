package dao

import "context"

func (s *skillDAO) Publist(ctx context.Context, offset int, limit int) ([]PubSkill, error) {
	var skills []PubSkill
	err := s.db.WithContext(ctx).Model(&PubSkill{}).
		Order("id desc").
		Offset(offset).Limit(limit).Find(&skills).Error
	return skills, err
}

func (s *skillDAO) PubCount(ctx context.Context) (int64, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&PubSkill{}).Count(&count).Error
	return count, err
}

func (s *skillDAO) PubInfo(ctx context.Context, id int64) (PubSkill, error) {
	var skill PubSkill
	err := s.db.WithContext(ctx).Model(&PubSkill{}).
		Where("id = ? ", id).First(&skill).Error
	return skill, err

}

func (s *skillDAO) PubLevels(ctx context.Context, id int64) ([]PubSkillLevel, error) {
	var skillLevels []PubSkillLevel
	err := s.db.WithContext(ctx).Model(&PubSkillLevel{}).Where("sid = ? ", id).Find(&skillLevels).Error
	return skillLevels, err
}

func (s *skillDAO) PubRequestInfo(ctx context.Context, id int64) ([]PubSKillPreRequest, error) {
	var reqs []PubSKillPreRequest
	err := s.db.WithContext(ctx).Model(&PubSKillPreRequest{}).Where("sid = ? ", id).Find(&reqs).Error
	return reqs, err
}
