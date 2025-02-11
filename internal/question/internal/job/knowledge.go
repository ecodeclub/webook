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

package job

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/question/internal/domain"
	"github.com/ecodeclub/webook/internal/question/internal/service"
	"github.com/gotomicro/ego/task/ejob"
)

// KnowledgeJobStarter 生成供 AI 平台使用的知识库数据
type KnowledgeJobStarter struct {
	batchSize int
	svc       service.Service
	baseDir   string
}

func NewKnowledgeJobStarter(svc service.Service, baseDir string) *KnowledgeJobStarter {
	// 默认十条一批
	return &KnowledgeJobStarter{
		svc:       svc,
		batchSize: 10,
		baseDir:   baseDir,
	}
}

// Start 不是好的写法，因为直接绕开了 service。
func (s *KnowledgeJobStarter) Start(ctx ejob.Context) error {
	// 准备一个文件
	writer, err := os.Create(fmt.Sprintf("%s/genknow_%d.csv", s.baseDir, time.Now().UnixMilli()))
	if err != nil {
		return err
	}
	defer writer.Close()
	return s.Export(ctx, writer)
}

// Export 返回导出的文件名字
func (s *KnowledgeJobStarter) Export(ctx ejob.Context, writer io.Writer) error {
	offset := 0
	limit := s.batchSize

	csvWriter := csv.NewWriter(writer)
	defer csvWriter.Flush()
	// 写入标题
	_ = csvWriter.Write([]string{"问题", "标签", "问题描述", "问题分析",
		"15K 答案", "25K 答案", "35K 答案"})
	for {
		cnt, err := s.Batch(ctx, offset, limit, csvWriter)
		if err != nil {
			return err
		}
		if cnt < limit {
			// 全部搞完了
			break
		}
		offset += cnt
	}
	return nil
}

// Batch 实现很不优雅，性能很差，但是我能少些很多代码。等后面性能瓶颈了再说
func (s *KnowledgeJobStarter) Batch(ctx ejob.Context, offset, limit int, writer *csv.Writer) (int, error) {
	batchCtx, cancel := context.WithTimeout(ctx.Ctx, time.Second*3)
	defer cancel()
	_, ques, err := s.svc.PubList(batchCtx, offset, limit)
	if err != nil {
		return 0, err
	}
	ids := slice.Map(ques, func(idx int, src domain.Question) int64 {
		return src.Id
	})
	// 优化性能
	for _, qid := range ids {
		detail, err := s.svc.PubDetail(batchCtx, qid)
		if err != nil {
			return 0, err
		}
		// 开始写入
		err = writer.Write([]string{
			detail.Title, strings.Join(detail.Labels, ";"), detail.Content,
			s.formatAnswer(detail.Answer.Analysis),
			s.formatAnswer(detail.Answer.Basic),
			s.formatAnswer(detail.Answer.Intermediate),
			s.formatAnswer(detail.Answer.Advanced),
		})
		if err != nil {
			return 0, err
		}
	}
	return len(ques), nil
}

func (s *KnowledgeJobStarter) formatAnswer(ans domain.AnswerElement) string {
	sb := strings.Builder{}
	sb.WriteString(ans.Content)
	sb.WriteByte('\n')
	sb.WriteString("关键字：" + ans.Keywords)
	sb.WriteByte('\n')
	sb.WriteString("引导点：" + ans.Guidance)
	sb.WriteByte('\n')
	sb.WriteString("亮点：" + ans.Highlight)
	sb.WriteByte('\n')
	sb.WriteString("速记口诀：" + ans.Shorthand)
	return sb.String()
}
