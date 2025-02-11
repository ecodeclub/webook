package dao

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm/clause"

	"gorm.io/gorm"
)

var ErrBaseFileNotFound = gorm.ErrRecordNotFound

// KnowledgeBaseDAO 知识库文件和业务的映射表
type KnowledgeBaseDAO interface {
	Save(ctx context.Context, file KnowledgeBaseFile) error
	GetInfo(ctx context.Context, platform, baseID, name string) (KnowledgeBaseFile, error)
}

type KnowledgeBaseFile struct {
	Id    int64  `gorm:"primaryKey;autoIncrement;"`
	Biz   string `gorm:"type:varchar(256);not null;comment:业务类型名"`
	BizID int64  `gorm:"type:varchar(256);not null;comment:业务id"`
	// 一个平台的一个知识库name是唯一的
	Name string `gorm:"type:varchar(256);uniqueIndex:name_platform_baseId"`
	// 文件id
	FileID string `gorm:"type:varchar(100)"`
	// 平台
	Platform string `gorm:"type:varchar(50);uniqueIndex:name_platform_baseId"`
	// 知识库id
	KnowledgeBaseID string `gorm:"type:varchar(100);uniqueIndex:name_platform_baseId"`
	// 其它字段按需添加
	Ctime int64
	Utime int64
}

type knowledgeBaseDAO struct {
	db *gorm.DB
}

func NewKnowledgeBaseDAO(db *gorm.DB) KnowledgeBaseDAO {
	return &knowledgeBaseDAO{
		db: db,
	}
}

func (r *knowledgeBaseDAO) Save(ctx context.Context, file KnowledgeBaseFile) error {
	now := time.Now().UnixMilli()
	file.Utime = now
	file.Ctime = now
	res := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{
				Name: "name",
			},
			{
				Name: "platform",
			},
			{
				Name: "repository_base_id",
			},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"file_id",
			"utime",
		}),
	}).Create(&file)
	return res.Error
}

func (r *knowledgeBaseDAO) GetInfo(ctx context.Context, platform, baseID, name string) (KnowledgeBaseFile, error) {
	var file KnowledgeBaseFile
	err := r.db.WithContext(ctx).Where("name = ? and platform = ? and knowledge_base_id = ?  ", name, platform, baseID).First(&file).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return file, ErrBaseFileNotFound
	}
	return file, nil
}
