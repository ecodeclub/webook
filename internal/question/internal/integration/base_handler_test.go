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

//go:build e2e

package integration

import (
	"fmt"
	"testing"

	"github.com/ecodeclub/webook/internal/question/internal/domain"

	"github.com/ecodeclub/webook/internal/question/internal/web"

	"github.com/ecodeclub/webook/internal/interactive"
	"github.com/ecodeclub/webook/internal/question/internal/repository/dao"
	"github.com/ego-component/egorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type BaseTestSuite struct {
	suite.Suite
	db *egorm.Component
}

func (s *BaseTestSuite) TearDownTest() {
	err := s.db.Exec("TRUNCATE TABLE `answer_elements`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE `questions`").Error
	require.NoError(s.T(), err)

	err = s.db.Exec("TRUNCATE TABLE `publish_answer_elements`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE `publish_questions`").Error
	require.NoError(s.T(), err)

	err = s.db.Exec("TRUNCATE TABLE `question_sets`").Error
	require.NoError(s.T(), err)

	err = s.db.Exec("TRUNCATE TABLE `question_set_questions`").Error
	require.NoError(s.T(), err)

	err = s.db.Exec("TRUNCATE TABLE `question_results`").Error
	require.NoError(s.T(), err)
}

// assertQuestionSetEqual 不比较 id
func (s *BaseTestSuite) assertQuestionSetEqual(t *testing.T, expect dao.QuestionSet, actual dao.QuestionSet) {
	assert.True(t, actual.Id > 0)
	assert.True(t, actual.Ctime > 0)
	assert.True(t, actual.Utime > 0)
	actual.Id = 0
	actual.Ctime = 0
	actual.Utime = 0
	assert.Equal(t, expect, actual)
}

// assertQuestion 不比较 id
func (s *BaseTestSuite) assertQuestion(t *testing.T, expect dao.Question, q dao.Question) {
	assert.True(t, q.Id > 0)
	assert.True(t, q.Ctime > 0)
	assert.True(t, q.Utime > 0)
	q.Id = 0
	q.Ctime = 0
	q.Utime = 0
	assert.Equal(t, expect, q)
}

func (s *BaseTestSuite) mockInteractive(biz string, id int64) interactive.Interactive {
	liked := id%2 == 1
	collected := id%2 == 0
	return interactive.Interactive{
		Biz:        biz,
		BizId:      id,
		ViewCnt:    int(id + 1),
		LikeCnt:    int(id + 2),
		CollectCnt: int(id + 3),
		Liked:      liked,
		Collected:  collected,
	}
}

func (s *BaseTestSuite) buildQuestion(id int64) dao.Question {
	return dao.Question{
		Id:      id,
		Uid:     uid,
		Biz:     domain.DefaultBiz,
		BizId:   id,
		Title:   fmt.Sprintf("标题%d", id),
		Content: fmt.Sprintf("内容%d", id),
		Ctime:   123 + id,
		Utime:   123 + id,
	}
}

func (s *BaseTestSuite) buildWebQuestion(id int64) web.Question {
	return web.Question{
		Id:      id,
		Biz:     domain.DefaultBiz,
		BizId:   id,
		Title:   fmt.Sprintf("标题%d", id),
		Content: fmt.Sprintf("内容%d", id),
		Utime:   123 + id,
	}
}

func (s *BaseTestSuite) buildDAOAnswerEle(
	qid int64,
	idx int,
	typ uint8) dao.AnswerElement {
	return dao.AnswerElement{
		Qid:       qid,
		Type:      typ,
		Content:   fmt.Sprintf("这是解析 %d", idx),
		Keywords:  fmt.Sprintf("关键字 %d", idx),
		Shorthand: fmt.Sprintf("快速记忆法 %d", idx),
		Highlight: fmt.Sprintf("亮点 %d", idx),
		Guidance:  fmt.Sprintf("引导点 %d", idx),
	}
}

func (s *BaseTestSuite) buildDomainAnswerEle(idx int, id int64) domain.AnswerElement {
	return domain.AnswerElement{
		Id:        id,
		Content:   fmt.Sprintf("这是解析 %d", idx),
		Keywords:  fmt.Sprintf("关键字 %d", idx),
		Shorthand: fmt.Sprintf("快速记忆法 %d", idx),
		Highlight: fmt.Sprintf("亮点 %d", idx),
		Guidance:  fmt.Sprintf("引导点 %d", idx),
	}
}

func (s *BaseTestSuite) buildAnswerEle(idx int64) web.AnswerElement {
	return web.AnswerElement{
		Content:   fmt.Sprintf("这是解析 %d", idx),
		Keywords:  fmt.Sprintf("关键字 %d", idx),
		Shorthand: fmt.Sprintf("快速记忆法 %d", idx),
		Highlight: fmt.Sprintf("亮点 %d", idx),
		Guidance:  fmt.Sprintf("引导点 %d", idx),
	}
}
