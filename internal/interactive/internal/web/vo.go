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

type CollectReq struct {
	Biz   string `json:"biz"`
	BizId int64  `json:"bizId"`
	// 目前还不支持收藏夹的功能。所以可以认为都是放到用户的默认收藏夹里面。
}

type Collection struct {
	// 如果传递了这个参数，那么就是更新，如果没有则是插入
	Id   int64  `json:"id"`
	Biz  string `json:"biz"`
	Name string `json:"name"`
}

type LikeReq struct {
	Biz   string `json:"biz"`
	BizId int64  `json:"bizId"`
	// true => 点赞
	// false => 取消点赞
	Liked bool `json:"liked"`
}
type ViewReq struct {
	Biz   string `json:"biz"`
	BizId int64  `json:"bizId"`
}

type GetCntReq struct {
	Biz   string `json:"biz"`
	BizId int64  `json:"bizId"`
}

type BatchGetCntReq struct {
	Biz    string  `json:"biz"`
	BizIds []int64 `json:"bizIds"`
}

type GetCntResp struct {
	CollectCnt int `json:"collectCnt"`
	LikeCnt    int `json:"likeCnt"`
	ViewCnt    int `json:"viewCnt"`
	// 是否收藏过
	Collected bool `json:"collected"`
	// 是否点赞过
	Liked bool `json:"liked"`
}

type Interactive struct {
	ID         int64 `json:"id"`
	CollectCnt int   `json:"collectCnt"`
	LikeCnt    int   `json:"likeCnt"`
	ViewCnt    int   `json:"viewCnt"`
	Liked      bool  `json:"liked"`
	Collected  bool  `json:"collected"`
}
type BatchGetCntResp struct {
	InteractiveMap map[int64]Interactive `json:"interactiveMap"`
}

type CollectionListReq struct {
	Biz    string `json:"biz"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

type IdReq struct {
	Id int64 `json:"id"`
}
type MoveCollectionReq struct {
	Biz   string `json:"biz"`
	BizId int64  `json:"bizId"`
	Cid   int64  `json:"cid"`
}
