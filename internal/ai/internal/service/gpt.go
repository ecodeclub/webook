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

package service

import "context"

//go:generate mockgen -source=./gpt.go -destination=../../mocks/gpt.mock.go -package=aimocks -typed=true GPTService
type GPTService interface {
	Invoke(ctx context.Context, req GPTRequest) (GPTResponse, error)
}

type GPTRequest struct {
	Biz   string
	Uid   int64
	Tid   string
	Input []string
}

type GPTResponse struct {
	Tokens int
	Amount int64
	Answer string
}
