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

package baguwen

import (
	"github.com/ecodeclub/webook/internal/question/internal/domain"
	"github.com/ecodeclub/webook/internal/question/internal/job"
	"github.com/ecodeclub/webook/internal/question/internal/service"
	"github.com/ecodeclub/webook/internal/question/internal/web"
)

type AdminHandler = web.AdminHandler
type AdminQuestionSetHandler = web.AdminQuestionSetHandler
type Handler = web.Handler
type QuestionSetHandler = web.QuestionSetHandler
type ExamineHandler = web.ExamineHandler
type KnowledgeBaseHandler = web.KnowledgeBaseHandler

type Service = service.Service
type QuestionSetService = service.QuestionSetService
type ExamService = service.ExamineService
type Question = domain.Question
type QuestionSet = domain.QuestionSet
type ExamResult = domain.ExamineResult
type ExamRes = domain.Result
type KnowledgeJobStarter = job.KnowledgeJobStarter
