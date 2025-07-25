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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/ecodeclub/ekit/iox"
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/comment/internal/integration/startup"
	"github.com/ecodeclub/webook/internal/comment/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/comment/internal/web"
	"github.com/ecodeclub/webook/internal/test"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/ecodeclub/webook/internal/user"
	usermocks "github.com/ecodeclub/webook/internal/user/mocks"
	"github.com/ego-component/egorm"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/econf"
	"github.com/gotomicro/ego/server/egin"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type HandlerTestSuite struct {
	suite.Suite
	server     *egin.Component
	db         *egorm.Component
	dao        dao.CommentDAO
	userModule *user.Module
}

const (
	testUID  = int64(12345)
	testUID2 = int64(12346)
	testUID3 = int64(12347)
)

func (s *HandlerTestSuite) SetupSuite() {
	s.db = testioc.InitDB()
	err := dao.InitTables(s.db)
	s.NoError(err)

	s.dao = dao.NewCommentGORMDAO(s.db)

	econf.Set("server", map[string]any{"contextTimeout": "1s"})
}

func (s *HandlerTestSuite) SetupTest() {
	// 创建Mock用户服务
	ctrl := gomock.NewController(s.T())
	mockUserSvc := usermocks.NewMockUserService(ctrl)

	s.setupMockUserService(mockUserSvc)

	// 创建用户模块
	s.userModule = &user.Module{Svc: mockUserSvc}

	// 初始化comment模块
	commentModule, err := startup.InitModule(s.userModule)
	s.NoError(err)

	// 设置服务器和路由
	server := egin.Load("server").Build()
	server.Use(func(ctx *gin.Context) {
		ctx.Set("_session", session.NewMemorySession(session.Claims{
			Uid: testUID,
			Data: map[string]string{
				"creator":   "true",
				"memberDDL": strconv.FormatInt(time.Now().Add(time.Hour).UnixMilli(), 10),
			},
		}))
	})

	commentModule.Hdl.MemberRoutes(server.Engine)
	s.server = server
}

func (s *HandlerTestSuite) setupMockUserService(mockUserSvc *usermocks.MockUserService) {
	testUsers := map[int64]user.User{
		testUID: {
			Id:       testUID,
			Nickname: "测试用户1",
			Avatar:   "avatar1.jpg",
		},
		testUID2: {
			Id:       testUID2,
			Nickname: "测试用户2",
			Avatar:   "avatar2.jpg",
		},
		testUID3: {
			Id:       testUID3,
			Nickname: "测试用户3",
			Avatar:   "avatar3.jpg",
		},
	}

	mockUserSvc.EXPECT().BatchProfile(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, ids []int64) ([]user.User, error) {
			users := make([]user.User, 0, len(ids))
			for _, id := range ids {
				if user, exists := testUsers[id]; exists {
					users = append(users, user)
				}
			}
			return users, nil
		}).AnyTimes()
}

func (s *HandlerTestSuite) TearDownTest() {
	s.NoError(s.db.Exec("TRUNCATE TABLE `comments`").Error)
}

// 生成唯一的业务ID，避免测试间冲突
func (s *HandlerTestSuite) getUniqueBizID() int64 {
	return time.Now().UnixNano()%1000000 + rand.Int63n(1000) + 10000
}

// 创建始祖评论，返回评论ID
func (s *HandlerTestSuite) createAncestorComment(biz string, bizID int64) int64 {
	comment := dao.Comment{
		Uid:        testUID,
		Biz:        biz,
		BizID:      bizID,
		ParentID:   0,
		AncestorID: 0,
		Content:    fmt.Sprintf("始祖评论_%d", time.Now().UnixNano()),
		Ctime:      time.Now().UnixMilli(),
		Utime:      time.Now().UnixMilli(),
	}

	err := s.db.Create(&comment).Error
	s.NoError(err)
	return comment.ID
}

// 创建回复评论，返回评论ID
func (s *HandlerTestSuite) createReplyComment(parentID, ancestorID int64, content string) int64 {
	comment := dao.Comment{
		Uid:        testUID2,
		ParentID:   parentID,
		AncestorID: ancestorID,
		Content:    content,
		Ctime:      time.Now().UnixMilli(),
		Utime:      time.Now().UnixMilli(),
	}

	err := s.db.Create(&comment).Error
	s.NoError(err)
	return comment.ID
}

// 辅助方法：查询评论列表
func (s *HandlerTestSuite) queryCommentList(req web.CommentListRequest) map[string]any {
	reqBody, _ := json.Marshal(req)
	httpReq := httptest.NewRequest(http.MethodPost, "/comment/list", bytes.NewReader(reqBody))
	httpReq.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	s.server.ServeHTTP(recorder, httpReq)

	// 添加调试信息
	if recorder.Code != 200 {
		s.T().Logf("Expected 200 but got %d, Response: %s", recorder.Code, recorder.Body.String())
	}

	s.Equal(200, recorder.Code)

	var result ginx.Result
	err := json.Unmarshal(recorder.Body.Bytes(), &result)
	s.NoError(err)

	return result.Data.(map[string]any)
}

func (s *HandlerTestSuite) TestCreateComment() {
	testCases := []struct {
		name       string
		setupData  func() (string, int64, int64) // biz, bizID, parentID
		reqContent string
		wantCode   int
		wantErr    bool
		verify     func(t *testing.T, commentID int64)
	}{
		{
			name: "TC01-创建始祖评论成功",
			setupData: func() (string, int64, int64) {
				return "article", s.getUniqueBizID(), 0
			},
			reqContent: "这是一个测试评论",
			wantCode:   200,
			verify: func(t *testing.T, commentID int64) {
				var comment dao.Comment
				err := s.db.First(&comment, commentID).Error
				s.NoError(err)
				s.Equal("这是一个测试评论", comment.Content)
				s.Equal(int64(0), comment.ParentID)
				s.Equal(int64(0), comment.AncestorID)
			},
		},
		{
			name: "TC02-创建一级回复成功",
			setupData: func() (string, int64, int64) {
				biz := "article"
				bizID := s.getUniqueBizID()
				parentID := s.createAncestorComment(biz, bizID)
				return biz, bizID, parentID
			},
			reqContent: "这是一个回复",
			wantCode:   200,
			verify: func(t *testing.T, commentID int64) {
				var comment dao.Comment
				err := s.db.First(&comment, commentID).Error
				s.NoError(err)
				s.NotEqual(int64(0), comment.ParentID)
				s.NotEqual(int64(0), comment.AncestorID)
			},
		},
		{
			name: "TC03-创建二级回复成功",
			setupData: func() (string, int64, int64) {
				biz := "course"
				bizID := s.getUniqueBizID()
				ancestorID := s.createAncestorComment(biz, bizID)
				replyID := s.createReplyComment(ancestorID, ancestorID, "一级回复")
				return biz, bizID, replyID
			},
			reqContent: "这是二级回复",
			wantCode:   200,
			verify: func(t *testing.T, commentID int64) {
				var comment dao.Comment
				err := s.db.First(&comment, commentID).Error
				s.NoError(err)
				s.NotEqual(int64(0), comment.AncestorID)
			},
		},
		{
			name: "TC04-创建多级深度回复成功",
			setupData: func() (string, int64, int64) {
				biz := "course"
				bizID := s.getUniqueBizID()
				ancestorID := s.createAncestorComment(biz, bizID)
				reply1ID := s.createReplyComment(ancestorID, ancestorID, "一级回复")
				reply2ID := s.createReplyComment(reply1ID, ancestorID, "二级回复")
				return biz, bizID, reply2ID
			},
			reqContent: "这是三级回复",
			wantCode:   200,
			verify: func(t *testing.T, commentID int64) {
				var comment dao.Comment
				err := s.db.First(&comment, commentID).Error
				s.NoError(err)
				s.NotEqual(int64(0), comment.AncestorID)
			},
		},
		{
			name: "TC13-无效ParentID回复失败",
			setupData: func() (string, int64, int64) {
				return "article", s.getUniqueBizID(), 99999999 // 不存在的ParentID
			},
			reqContent: "回复不存在的评论",
			wantCode:   500,
			wantErr:    true,
		},
		{
			name: "TC15-空内容评论失败",
			setupData: func() (string, int64, int64) {
				return "article", s.getUniqueBizID(), 0
			},
			reqContent: "",
			wantCode:   500,
			wantErr:    true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			biz, bizID, parentID := tc.setupData()

			req := web.CommentRequest{
				Comment: web.Comment{
					Biz:      biz,
					BizID:    bizID,
					ParentID: parentID,
					Content:  tc.reqContent,
				},
			}

			httpReq, err := http.NewRequest(http.MethodPost, "/comment/", iox.NewJSONReader(req))
			httpReq.Header.Set("Content-Type", "application/json")
			s.NoError(err)
			recorder := test.NewJSONResponseRecorder[any]()
			s.server.ServeHTTP(recorder, httpReq)

			// 添加调试信息
			if recorder.Code != tc.wantCode {
				s.T().Logf("Expected %d but got %d, Response: %s", tc.wantCode, recorder.Code, recorder.Body.String())
			}

			s.Equal(tc.wantCode, recorder.Code)

			if !tc.wantErr && recorder.Code == 200 {
				var result ginx.Result
				err := json.Unmarshal(recorder.Body.Bytes(), &result)
				s.NoError(err)

				commentID, ok := result.Data.(float64) // JSON数字解析为float64
				s.True(ok)
				s.True(commentID > 0)

				if tc.verify != nil {
					tc.verify(s.T(), int64(commentID))
				}
			}
		})
	}
}

func (s *HandlerTestSuite) TestCommentList() {
	testCases := []struct {
		name      string
		setupData func() (string, int64, []int64) // biz, bizID, commentIDs
		req       web.CommentListRequest
		wantCode  int
		verify    func(t *testing.T, result map[string]any)
	}{
		{
			name: "TC05-始祖评论分页查询按时间倒序",
			setupData: func() (string, int64, []int64) {
				biz := "article"
				bizID := s.getUniqueBizID()

				ids := make([]int64, 3)
				for i := 0; i < 3; i++ {
					ids[i] = s.createAncestorComment(biz, bizID)
					time.Sleep(time.Millisecond)
				}
				return biz, bizID, ids
			},
			req: web.CommentListRequest{
				Limit:     10,
				MaxSubCnt: 0,
			},
			wantCode: 200,
			verify: func(t *testing.T, result map[string]any) {
				list := result["list"].([]any)
				total := int(result["total"].(float64))

				s.Equal(3, len(list))
				s.Equal(3, total)

				if len(list) >= 2 {
					first := list[0].(map[string]any)
					second := list[1].(map[string]any)
					firstTime := int64(first["utime"].(float64))
					secondTime := int64(second["utime"].(float64))
					s.True(firstTime >= secondTime) // 倒序
				}
			},
		},
		{
			name: "TC07-带子评论预加载的分页查询",
			setupData: func() (string, int64, []int64) {
				biz := "article"
				bizID := s.getUniqueBizID()

				ancestorID := s.createAncestorComment(biz, bizID)

				for i := 0; i < 5; i++ {
					s.createReplyComment(ancestorID, ancestorID, fmt.Sprintf("子评论%d", i+1))
				}

				return biz, bizID, []int64{ancestorID}
			},
			req: web.CommentListRequest{
				Limit:     10,
				MaxSubCnt: 3,
			},
			wantCode: 200,
			verify: func(t *testing.T, result map[string]any) {
				list := result["list"].([]any)
				s.Equal(1, len(list))

				comment := list[0].(map[string]any)
				replies := comment["replies"].([]any)
				s.Equal(3, len(replies)) // 只预加载了3个

				// TC09-验证评论用户信息填充
				user := comment["user"].(map[string]any)
				s.Equal("测试用户1", user["nickname"])
				s.Equal("avatar1.jpg", user["avatar"])
			},
		},
		{
			name: "TC08-边界分页测试",
			setupData: func() (string, int64, []int64) {
				biz := "article"
				bizID := s.getUniqueBizID()

				ids := make([]int64, 5)
				for i := 0; i < 5; i++ {
					ids[i] = s.createAncestorComment(biz, bizID)
					time.Sleep(time.Millisecond)
				}
				return biz, bizID, ids
			},
			req: web.CommentListRequest{
				Limit:     3,
				MaxSubCnt: 0,
			},
			wantCode: 200,
			verify: func(t *testing.T, result map[string]any) {
				list := result["list"].([]any)
				total := int(result["total"].(float64))

				s.Equal(3, len(list)) // 分页限制
				s.Equal(5, total)     // 总数
			},
		},
		{
			name: "TC14-查询不存在的业务资源",
			setupData: func() (string, int64, []int64) {
				return "article", s.getUniqueBizID(), nil
			},
			req: web.CommentListRequest{
				Limit:     10,
				MaxSubCnt: 0,
			},
			wantCode: 200,
			verify: func(t *testing.T, result map[string]any) {
				list := result["list"].([]any)
				total := int(result["total"].(float64))

				s.Equal(0, len(list))
				s.Equal(0, total)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			biz, bizID, _ := tc.setupData()

			tc.req.Biz = biz
			tc.req.BizID = bizID

			httpReq, err := http.NewRequest(http.MethodPost, "/comment/list", iox.NewJSONReader(tc.req))
			httpReq.Header.Set("Content-Type", "application/json")
			s.NoError(err)
			recorder := test.NewJSONResponseRecorder[any]()
			s.server.ServeHTTP(recorder, httpReq)

			s.Equal(tc.wantCode, recorder.Code)

			if recorder.Code == 200 {
				var result ginx.Result
				err := json.Unmarshal(recorder.Body.Bytes(), &result)
				s.NoError(err)

				data := result.Data.(map[string]any)
				if tc.verify != nil {
					tc.verify(s.T(), data)
				}
			}
		})
	}
}

func (s *HandlerTestSuite) TestGetReplies() {
	testCases := []struct {
		name      string
		setupData func() (int64, []int64) // ancestorID, replyIDs
		req       web.GetRepliesRequest
		wantCode  int
		verify    func(t *testing.T, result map[string]any)
	}{
		{
			name: "TC06-回复分页查询按时间正序",
			setupData: func() (int64, []int64) {
				biz := "article"
				bizID := s.getUniqueBizID()
				ancestorID := s.createAncestorComment(biz, bizID)

				replyIDs := make([]int64, 0)
				for i := 0; i < 2; i++ {
					reply1ID := s.createReplyComment(ancestorID, ancestorID, fmt.Sprintf("一级回复%d", i+1))
					replyIDs = append(replyIDs, reply1ID)

					for j := 0; j < 2; j++ {
						reply2ID := s.createReplyComment(reply1ID, ancestorID, fmt.Sprintf("二级回复%d-%d", i+1, j+1))
						replyIDs = append(replyIDs, reply2ID)
					}
				}

				return ancestorID, replyIDs
			},
			req: web.GetRepliesRequest{
				MaxID: 0,
				Limit: 10,
			},
			wantCode: 200,
			verify: func(t *testing.T, result map[string]any) {
				list := result["list"].([]any)
				total := int(result["total"].(float64))

				s.Equal(6, len(list)) // 2个一级 + 4个二级
				s.Equal(6, total)

				// 验证按时间正序排列（与始祖评论查询相反）
				if len(list) >= 2 {
					first := list[0].(map[string]any)
					second := list[1].(map[string]any)
					firstTime := int64(first["utime"].(float64))
					secondTime := int64(second["utime"].(float64))
					s.True(firstTime <= secondTime) // 正序
				}

				// TC10-验证回复用户信息填充
				if len(list) > 0 {
					reply := list[0].(map[string]any)
					user := reply["user"].(map[string]any)
					s.Equal("测试用户2", user["nickname"])
					s.Equal("avatar2.jpg", user["avatar"])
				}
			},
		},
		{
			name: "分页查询回复",
			setupData: func() (int64, []int64) {
				biz := "article"
				bizID := s.getUniqueBizID()
				ancestorID := s.createAncestorComment(biz, bizID)

				replyIDs := make([]int64, 5)
				for i := 0; i < 5; i++ {
					replyIDs[i] = s.createReplyComment(ancestorID, ancestorID, fmt.Sprintf("回复%d", i+1))
					time.Sleep(time.Millisecond)
				}

				return ancestorID, replyIDs
			},
			req: web.GetRepliesRequest{
				MaxID: 0,
				Limit: 3,
			},
			wantCode: 200,
			verify: func(t *testing.T, result map[string]any) {
				list := result["list"].([]any)
				total := int(result["total"].(float64))

				s.Equal(3, len(list)) // 分页限制
				s.Equal(5, total)     // 总数
			},
		},
		{
			name: "查询不存在的始祖评论",
			setupData: func() (int64, []int64) {
				return 99999999, nil // 不存在的ID
			},
			req: web.GetRepliesRequest{
				MaxID: 0,
				Limit: 10,
			},
			wantCode: 200,
			verify: func(t *testing.T, result map[string]any) {
				list := result["list"].([]any)
				total := int(result["total"].(float64))

				s.Equal(0, len(list))
				s.Equal(0, total)
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			ancestorID, _ := tc.setupData()

			tc.req.AncestorID = ancestorID

			httpReq, err := http.NewRequest(http.MethodPost, "/comment/replies", iox.NewJSONReader(tc.req))
			httpReq.Header.Set("Content-Type", "application/json")
			s.NoError(err)
			recorder := test.NewJSONResponseRecorder[any]()

			s.server.ServeHTTP(recorder, httpReq)

			s.Equal(tc.wantCode, recorder.Code)

			if recorder.Code == 200 {
				var result ginx.Result
				err := json.Unmarshal(recorder.Body.Bytes(), &result)
				s.NoError(err)

				data := result.Data.(map[string]any)
				if tc.verify != nil {
					tc.verify(s.T(), data)
				}
			}
		})
	}
}

func (s *HandlerTestSuite) TestBusinessIsolation() {
	testCases := []struct {
		name   string
		setup  func() (string, int64, string, int64) // biz1, bizID1, biz2, bizID2
		verify func(t *testing.T, biz1 string, bizID1 int64, biz2 string, bizID2 int64)
	}{
		{
			name: "TC11-不同Biz类型隔离",
			setup: func() (string, int64, string, int64) {
				articleBizID := s.getUniqueBizID()
				courseBizID := s.getUniqueBizID()

				s.createAncestorComment("article", articleBizID)
				s.createAncestorComment("article", articleBizID)
				s.createAncestorComment("course", courseBizID)

				return "article", articleBizID, "course", courseBizID
			},
			verify: func(t *testing.T, biz1 string, bizID1 int64, biz2 string, bizID2 int64) {
				req1 := web.CommentListRequest{
					Biz:   biz1,
					BizID: bizID1,
					Limit: 10,
				}

				req2 := web.CommentListRequest{
					Biz:   biz2,
					BizID: bizID2,
					Limit: 10,
				}

				result1 := s.queryCommentList(req1)
				list1 := result1["list"].([]any)
				s.Equal(2, len(list1))

				result2 := s.queryCommentList(req2)
				list2 := result2["list"].([]any)
				s.Equal(1, len(list2))
			},
		},
		{
			name: "TC12-相同Biz不同BizID隔离",
			setup: func() (string, int64, string, int64) {
				biz := "article"
				bizID1 := s.getUniqueBizID()
				bizID2 := s.getUniqueBizID()

				s.createAncestorComment(biz, bizID1)
				s.createAncestorComment(biz, bizID1)
				s.createAncestorComment(biz, bizID2)

				return biz, bizID1, biz, bizID2
			},
			verify: func(t *testing.T, biz1 string, bizID1 int64, biz2 string, bizID2 int64) {
				req1 := web.CommentListRequest{
					Biz:   biz1,
					BizID: bizID1,
					Limit: 10,
				}

				req2 := web.CommentListRequest{
					Biz:   biz2,
					BizID: bizID2,
					Limit: 10,
				}

				result1 := s.queryCommentList(req1)
				list1 := result1["list"].([]any)
				s.Equal(2, len(list1))

				result2 := s.queryCommentList(req2)
				list2 := result2["list"].([]any)
				s.Equal(1, len(list2))
			},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			biz1, bizID1, biz2, bizID2 := tc.setup()
			tc.verify(s.T(), biz1, bizID1, biz2, bizID2)
		})
	}
}

func TestHandler(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}
