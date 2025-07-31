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

package domain

type MaterialStatus string

const (
	MaterialStatusInit     MaterialStatus = "INIT"
	MaterialStatusAccepted MaterialStatus = "ACCEPTED"
)

func (m MaterialStatus) String() string {
	return string(m)
}

type Material struct {
	ID        int64
	Uid       int64
	AudioURL  string
	ResumeURL string
	Remark    string
	Status    MaterialStatus
	Ctime     int64
	Utime     int64
}
