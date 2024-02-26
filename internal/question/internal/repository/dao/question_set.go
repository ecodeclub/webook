package dao

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ego-component/egorm"
	"github.com/go-sql-driver/mysql"
	"github.com/gotomicro/ego/core/elog"
	"gorm.io/gorm"
)

var (
	ErrDuplicatedQuestionID = errors.New("问题ID重复")
)

type QuestionSetDAO interface {
	Create(ctx context.Context, qs QuestionSet) (int64, error)
	GetByID(ctx context.Context, id int64) (QuestionSet, error)
	// List(ctx context.Context, offset int, limit int, uid int64) ([]Question, error)

	GetQuestionsByID(ctx context.Context, id int64) ([]Question, error)
	UpdateQuestionsByID(ctx context.Context, id int64, qs []Question) error
	AddQuestionsByID(ctx context.Context, id int64, qs []Question) error
	DeleteQuestionsByID(ctx context.Context, id int64, qs []Question) error

	// Count(ctx context.Context, uid int64) (int64, error)
	//
	// Sync(ctx context.Context, que Question, eles []AnswerElement) (int64, error)
	//
	// // 线上库 API
	// PubList(ctx context.Context, offset int, limit int) ([]PublishQuestion, error)
	// PubCount(ctx context.Context) (int64, error)
	// GetPubByID(ctx context.Context, qid int64) (PublishQuestion, []PublishAnswerElement, error)
}

type GORMQuestionSetDAO struct {
	db *egorm.Component
}

func NewGORMQuestionSetDAO(db *egorm.Component) QuestionSetDAO {
	return &GORMQuestionSetDAO{db: db}
}

func (g *GORMQuestionSetDAO) Create(ctx context.Context, qs QuestionSet) (int64, error) {
	now := time.Now().UnixMilli()
	qs.Ctime, qs.Utime = now, now
	err := g.db.WithContext(ctx).Create(&qs).Error
	return qs.Id, err
}

func (g *GORMQuestionSetDAO) GetByID(ctx context.Context, id int64) (QuestionSet, error) {
	var qs QuestionSet
	err := g.db.WithContext(ctx).First(&qs, "id = ?", id).Error
	return qs, err
}

func (g *GORMQuestionSetDAO) GetQuestionsByID(ctx context.Context, id int64) ([]Question, error) {
	var q []Question
	err := g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var qsq []QuestionSetQuestion
		if err := tx.WithContext(ctx).Find(&qsq, "question_set_id = ?", id).Error; err != nil {
			return err
		}
		questionIDs := make([]int64, len(qsq))
		for i := range qsq {
			questionIDs[i] = qsq[i].QuestionID
		}
		return tx.WithContext(ctx).Where("id IN ?", questionIDs).Order("id ASC").Find(&q).Error
	})
	return q, err
}

func (g *GORMQuestionSetDAO) UpdateQuestionsByID(ctx context.Context, id int64, questions []Question) error {
	return g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		log.Println("UpdateQuestionsByID Invoked!")
		// 获取题集中的目标题目集合
		targetQuestionIDs := make([]int64, len(questions))
		for i := range questions {
			targetQuestionIDs[i] = questions[i].Id
		}

		log.Printf("targetQuestionIDs = %#v\n", targetQuestionIDs)

		// 检查目标问题ID是否合法
		var count int64
		if err := tx.WithContext(ctx).Model(&Question{}).Where("id IN ?", targetQuestionIDs).Count(&count).Error; err != nil {
			return err
		}
		if int64(len(questions)) != count {
			return fmt.Errorf("问题ID非法")
		}

		log.Println("UpdateQuestionsByID Invoked!3")

		// 题集中现有的问题ID集合
		var currentQuestions []QuestionSetQuestion
		if err := tx.WithContext(ctx).Find(&currentQuestions, "question_set_id = ?", id).Error; err != nil {
			log.Println("err >>>>>", err)
			return err
		}
		currentQuestionIDs := make([]int64, len(currentQuestions))
		for i := range currentQuestions {
			currentQuestionIDs[i] = currentQuestions[i].QuestionID
		}

		log.Printf("currentQuestionIDs = %#v\n", currentQuestionIDs)

		log.Println("UpdateQuestionsByID Invoked!4")

		// 在当前问题集合中但不在目标问题集合中 —— 删除问题
		needDelete := slice.DiffSet(currentQuestionIDs, targetQuestionIDs)
		if len(needDelete) > 0 {
			err := tx.WithContext(ctx).Where("question_set_id = ? AND question_id IN ?", id, needDelete).Delete(&QuestionSetQuestion{}).Error
			if err != nil {
				return err
			}
		}

		log.Printf("needDelete = %#v\n", needDelete)

		// 在目标问题集合中但不在当前问题集合中 —— 新增问题
		needCreate := slice.DiffSet(targetQuestionIDs, currentQuestionIDs)
		if len(needCreate) > 0 {
			now := time.Now().UnixMilli()
			var newQuestions []QuestionSetQuestion
			for i := range needCreate {
				newQuestions = append(newQuestions, QuestionSetQuestion{
					QuestionSetID: id,
					QuestionID:    needCreate[i],
					Ctime:         now,
					Utime:         now,
				})
			}
			if err := tx.WithContext(ctx).Create(&newQuestions).Error; err != nil {
				return err
			}
		}
		log.Printf("needCreate = %#v\n", needCreate)
		log.Println("UpdateQuestionsByID Invoked!5")
		return nil
	})
}

func (g *GORMQuestionSetDAO) AddQuestionsByID(ctx context.Context, id int64, questions []Question) error {
	return g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		questionIDs, err := g.getQuestionIDs(ctx, tx, questions)
		if err != nil {
			return err
		}

		now := time.Now().UnixMilli()
		newQuestions := make([]QuestionSetQuestion, 0, len(questionIDs))
		for i := range questionIDs {
			newQuestions = append(newQuestions, QuestionSetQuestion{
				QuestionSetID: id,
				QuestionID:    questionIDs[i],
				Ctime:         now,
				Utime:         now,
			})
		}
		err = tx.WithContext(ctx).Create(&newQuestions).Error
		if g.isMySQLUniqueIndexError(err) {
			return fmt.Errorf("%w", ErrDuplicatedQuestionID)
		}
		return err
	})
}

func (g *GORMQuestionSetDAO) isMySQLUniqueIndexError(err error) bool {
	me := new(mysql.MySQLError)
	if ok := errors.As(err, &me); ok {
		elog.Error("mysql", elog.FieldValue(fmt.Sprintf("%#v", err)))
		const uniqueIndexErrNo uint16 = 1062
		return me.Number == uniqueIndexErrNo
	}
	return false
}

func (g *GORMQuestionSetDAO) getQuestionIDs(ctx context.Context, tx *gorm.DB, questions []Question) ([]int64, error) {
	questionIDs := make([]int64, len(questions))
	for i := range questions {
		questionIDs[i] = questions[i].Id
	}

	// 检查目标问题ID是否合法
	var count int64
	if err := tx.WithContext(ctx).Model(&Question{}).Where("id IN ?", questionIDs).Count(&count).Error; err != nil {
		return nil, err
	}
	if int64(len(questions)) != count {
		return nil, fmt.Errorf("问题ID非法")
	}
	return questionIDs, nil
}

func (g *GORMQuestionSetDAO) DeleteQuestionsByID(ctx context.Context, id int64, questions []Question) error {
	return g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		questionIDs, err := g.getQuestionIDs(ctx, tx, questions)
		if err != nil {
			return err
		}
		return tx.WithContext(ctx).Where("question_set_id = ? AND question_id IN ?", id, questionIDs).Delete(&QuestionSetQuestion{}).Error
	})
}
