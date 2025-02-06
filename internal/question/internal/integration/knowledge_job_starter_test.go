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
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/ecodeclub/ginx/session"

	"github.com/ecodeclub/webook/internal/member"

	"github.com/ecodeclub/webook/internal/ai"

	"github.com/ecodeclub/webook/internal/interactive"
	"github.com/ecodeclub/webook/internal/permission"
	"github.com/ecodeclub/webook/internal/question/internal/domain"
	"github.com/ecodeclub/webook/internal/question/internal/integration/startup"
	"github.com/ecodeclub/webook/internal/question/internal/job"
	"github.com/ecodeclub/webook/internal/question/internal/repository/dao"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/gotomicro/ego/task/ejob"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// manual 这个手动运行
type KnowledgeJobStarterTestSuite struct {
	BaseTestSuite
	starter *job.KnowledgeJobStarter
	dao     dao.QuestionDAO
}

func (s *KnowledgeJobStarterTestSuite) SetupSuite() {
	module, err := startup.InitModule(nil, nil, &interactive.Module{}, &permission.Module{}, &ai.Module{},
		session.DefaultProvider(),
		&member.Module{})
	require.NoError(s.T(), err)
	s.starter = module.KnowledgeJobStarter
	s.db = testioc.InitDB()
	s.dao = dao.NewGORMQuestionDAO(s.db)
}

// 单独测试 TestBatch
func (s *KnowledgeJobStarterTestSuite) TestExport() {
	// 插入一些数据
	s.initWholeQuestion(1)
	s.initWholeQuestion(2)
	file, err := os.CreateTemp("", "gen_file")
	require.NoError(s.T(), err)
	defer func() {
		file.Close()
	}()

	err = s.starter.Export(ejob.Context{Ctx: context.Background()}, file)
	assert.NoError(s.T(), err)
	// 重置到文件开始位置
	_, err = file.Seek(0, 0)
	require.NoError(s.T(), err)
	all, err := io.ReadAll(file)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), ``, string(all))
}

func (s *KnowledgeJobStarterTestSuite) initWholeQuestion(id int64) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	_, err := s.dao.Sync(ctx, dao.Question{
		Id:      id,
		Uid:     uid,
		Title:   fmt.Sprintf("标题%d", id),
		Content: fmt.Sprintf("内容%d", id),
		Biz:     "question",
		BizId:   id*10 + 1,
		Status:  domain.PublishedStatus.ToUint8(),
		Ctime:   123,
		Utime:   123,
	}, []dao.AnswerElement{
		s.buildDAOAnswerEle(id, 1, dao.AnswerElementTypeAnalysis),
		s.buildDAOAnswerEle(id, 2, dao.AnswerElementTypeBasic),
		s.buildDAOAnswerEle(id, 3, dao.AnswerElementTypeIntermedia),
		s.buildDAOAnswerEle(id, 41, dao.AnswerElementTypeAdvanced),
	})
	assert.NoError(s.T(), err)
}

// 这个要手动运行来核对 excel 的格式
//func TestKnowledgeJobStarter(t *testing.T) {
//	suite.Run(t, new(KnowledgeJobStarterTestSuite))
//}
