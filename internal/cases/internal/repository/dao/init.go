package dao

import "github.com/ego-component/egorm"

func InitTables(db *egorm.Component) error {
	return db.AutoMigrate(
		&Case{},
		&PublishCase{},
		&CaseSet{},
		&CaseSetCase{},
		&CaseResult{},
		&CaseExamineRecord{},
		&CaseExamineRecord{},
		&CaseResult{},
	)
}
