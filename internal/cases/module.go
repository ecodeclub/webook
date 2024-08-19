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

package cases

import (
	"github.com/ecodeclub/webook/internal/cases/internal/domain"
	"github.com/ecodeclub/webook/internal/cases/internal/service"
	"github.com/ecodeclub/webook/internal/cases/internal/web"
)

type Module struct {
	Svc             Service
	SetSvc          SetService
	ExamService     ExamService
	Hdl             *Handler
	AdminSetHandler *AdminCaseSetHandler
	ExamineHdl      *ExamineHandler
	CsHdl           *CaseSetHandler
}

type Handler = web.Handler
type Service = service.Service
type SetService = service.CaseSetService
type Case = domain.Case
type CaseSet = domain.CaseSet
type AdminCaseSetHandler = web.AdminCaseSetHandler
type ExamineHandler = web.ExamineHandler
type CaseSetHandler = web.CaseSetHandler
type ExamService = service.ExamineService
type ExamineCaseResult = domain.ExamineCaseResult
type CaseResult = domain.CaseResult
