// Copyright 2023 ecodeclub
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ecodeclub/webook/internal/question/internal/domain"
	"github.com/pkg/errors"

	"github.com/ecodeclub/ecache"
)

type QuestionECache struct {
	ec ecache.Cache
}

var (
	ErrQuestionNotFound = errors.New("问题没找到")
)

const (
	expiration = 24 * time.Hour
)

func NewQuestionECache(ec ecache.Cache) QuestionCache {
	return &QuestionECache{
		ec: &ecache.NamespaceCache{
			Namespace: "question:",
			C:         ec,
		},
	}
}
func (q *QuestionECache) SetQuestion(ctx context.Context, question domain.Question) error {
	questionByte, err := json.Marshal(question)
	if err != nil {
		return errors.Wrap(err, "序列化问题失败")
	}
	return q.ec.Set(ctx, q.questionKey(question.Id), string(questionByte), expiration)
}

func (q *QuestionECache) GetQuestion(ctx context.Context, id int64) (domain.Question, error) {
	qVal := q.ec.Get(ctx, q.questionKey(id))
	if qVal.KeyNotFound() {
		return domain.Question{}, ErrQuestionNotFound
	}
	if qVal.Err != nil {
		return domain.Question{}, errors.Wrap(qVal.Err, "查询缓存出错")
	}

	var question domain.Question
	err := json.Unmarshal([]byte(qVal.Val.(string)), &question)
	if err != nil {
		return domain.Question{}, errors.Wrap(err, "反序列化问题失败")
	}
	return question, nil
}

func (q *QuestionECache) SetQuestions(ctx context.Context, biz string, questions []domain.Question) error {
	questionsByte, err := json.Marshal(questions)
	if err != nil {
		return errors.Wrap(err, "序列化问题失败")
	}
	return q.ec.Set(ctx, q.questionListKey(biz), string(questionsByte), expiration)
}

func (q *QuestionECache) GetQuestions(ctx context.Context, biz string) ([]domain.Question, error) {
	key := q.questionListKey(biz)
	qVal := q.ec.Get(ctx, key)
	if qVal.KeyNotFound() {
		return nil, ErrQuestionNotFound
	}
	if qVal.Err != nil {
		return nil, errors.Wrap(qVal.Err, "查询缓存出错")
	}

	var questions []domain.Question
	err := json.Unmarshal([]byte(qVal.Val.(string)), &questions)
	if err != nil {
		return nil, errors.Wrap(err, "反序列化问题失败")
	}
	return questions, nil
}

func (q *QuestionECache) GetTotal(ctx context.Context, biz string) (int64, error) {
	return q.ec.Get(ctx, q.totalKey(biz)).AsInt64()
}

func (q *QuestionECache) SetTotal(ctx context.Context, biz string, total int64) error {
	// 设置更久的过期时间都可以，毕竟很少更新题库
	return q.ec.Set(ctx, q.totalKey(biz), total, time.Minute*30)
}

func (q *QuestionECache) DelQuestion(ctx context.Context, id int64) error {
	_, err := q.ec.Delete(ctx, q.questionKey(id))
	return err
}

// 注意 Namespace 设置
func (q *QuestionECache) totalKey(biz string) string {
	return fmt.Sprintf("total:%s", biz)
}

func (q *QuestionECache) questionKey(id int64) string {
	return fmt.Sprintf("publish:%d", id)
}

func (q *QuestionECache) questionListKey(biz string) string {
	return fmt.Sprintf("list:%s", biz)
}
