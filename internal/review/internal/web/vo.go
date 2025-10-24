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

package web

import (
	"github.com/ecodeclub/webook/internal/company"
	"github.com/ecodeclub/webook/internal/interactive"
	"github.com/ecodeclub/webook/internal/review/internal/domain"
)

type ReviewSaveReq struct {
	Review Review `json:"review"`
}

type Review struct {
	ID          int64       `json:"id,omitempty"`
	Title       string      `json:"title,omitempty"`
	Desc        string      `json:"desc,omitempty"`
	Labels      []string    `json:"labels,omitempty"`
	JD          string      `json:"jd,omitempty"`
	Content     string      `json:"content,omitempty"`
	Resume      string      `json:"resume,omitempty"`
	Status      uint8       `json:"status,omitempty"`
	Utime       int64       `json:"utime,omitempty"`
	Interactive Interactive `json:"interactive,omitempty"`
	Company     Company     `json:"company,omitempty"`
}
type Company struct {
	ID   int64  `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}
type Interactive struct {
	CollectCnt int  `json:"collectCnt"`
	LikeCnt    int  `json:"likeCnt"`
	ViewCnt    int  `json:"viewCnt"`
	Liked      bool `json:"liked"`
	Collected  bool `json:"collected"`
}
type ReviewListResp struct {
	Total int64    `json:"total"`
	List  []Review `json:"list"`
}

func newCompleteReview(re domain.Review,
	intr interactive.Interactive,
	company company.Company,
) Review {
	review := newReviewWithCompany(re, company)
	review.Interactive = newInteractive(intr)
	return review
}
func newReviewWithCompany(re domain.Review, company company.Company) Review {
	review := newReview(re)
	review.Company = Company{
		ID:   company.ID,
		Name: company.Name,
	}
	return review
}

func newReview(re domain.Review) Review {
	return Review{
		ID:      re.ID,
		JD:      re.JD,
		Title:   re.Title,
		Desc:    re.Desc,
		Labels:  re.Labels,
		Content: re.Content,
		Resume:  re.Resume,
		Status:  re.Status.ToUint8(),
		Utime:   re.Utime,
	}
}

func (r Review) toDomain() domain.Review {
	return domain.Review{
		ID:      r.ID,
		Title:   r.Title,
		Desc:    r.Desc,
		Labels:  r.Labels,
		JD:      r.JD,
		Content: r.Content,
		Resume:  r.Resume,
		Status:  domain.ReviewStatus(r.Status),
		Utime:   r.Utime,
		Company: domain.Company{
			ID: r.Company.ID,
		},
	}
}

type DetailReq struct {
	ID int64 `json:"id,omitempty"`
}
type Page struct {
	Offset int `json:"offset,omitempty"`
	Limit  int `json:"limit,omitempty"`
}

func newInteractive(intr interactive.Interactive) Interactive {
	return Interactive{
		CollectCnt: intr.CollectCnt,
		ViewCnt:    intr.ViewCnt,
		LikeCnt:    intr.LikeCnt,
		Liked:      intr.Liked,
		Collected:  intr.Collected,
	}
}
