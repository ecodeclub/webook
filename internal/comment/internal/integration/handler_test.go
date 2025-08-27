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
	"database/sql"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"testing"
	"time"

	"github.com/ecodeclub/ekit/iox"
	"github.com/ecodeclub/ginx/session"
	evtmocks "github.com/ecodeclub/webook/internal/comment/internal/event/mocks"
	"github.com/ecodeclub/webook/internal/comment/internal/repository"
	"github.com/ecodeclub/webook/internal/comment/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/comment/internal/service"
	"github.com/ecodeclub/webook/internal/comment/internal/web"
	"github.com/ecodeclub/webook/internal/notification/event"
	"github.com/ecodeclub/webook/internal/test"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/ecodeclub/webook/internal/user"
	usermocks "github.com/ecodeclub/webook/internal/user/mocks"
	"github.com/ego-component/egorm"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/econf"
	"github.com/gotomicro/ego/server/egin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type HandlerTestSuite struct {
	suite.Suite
	server *egin.Component
	db     *egorm.Component
	dao    dao.CommentDAO
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
}

func (s *HandlerTestSuite) newGinServer(handler *web.Handler, uid int64) *egin.Component {
	econf.Set("server", map[string]any{"contextTimeout": "1s"})
	server := egin.Load("server").Build()
	server.Use(func(ctx *gin.Context) {
		ctx.Set("_session", session.NewMemorySession(session.Claims{
			Uid: uid,
		}))
	})
	handler.MemberRoutes(server.Engine)
	return server
}

func (s *HandlerTestSuite) TearDownSuite() {
	s.NoError(s.db.Exec("TRUNCATE TABLE `comments`").Error)
}

// 生成唯一的业务ID，避免测试间冲突
func (s *HandlerTestSuite) getUniqueBizID() int64 {
	return time.Now().UnixNano()%1000000 + rand.Int63n(1000) + 10000
}

// 创建始祖评论，返回评论ID
func (s *HandlerTestSuite) createAncestorComment(biz string, bizID int64) int64 {
	cmt := dao.Comment{
		Uid:        testUID,
		Biz:        biz,
		BizID:      bizID,
		ParentID:   sql.Null[int64]{V: 0, Valid: false},
		AncestorID: sql.Null[int64]{V: 0, Valid: false},
		Content:    fmt.Sprintf("始祖评论_%d", time.Now().UnixNano()),
		Ctime:      time.Now().UnixMilli(),
		Utime:      time.Now().UnixMilli(),
	}

	err := s.db.Create(&cmt).Error
	s.NoError(err)
	return cmt.ID
}

// 创建回复评论，返回评论ID
func (s *HandlerTestSuite) createReplyComment(parentID, ancestorID int64, content string) int64 {
	cmt := dao.Comment{
		Uid:        testUID2,
		ParentID:   sql.Null[int64]{V: parentID, Valid: true},
		AncestorID: sql.Null[int64]{V: ancestorID, Valid: true},
		Content:    content,
		Ctime:      time.Now().UnixMilli(),
		Utime:      time.Now().UnixMilli(),
	}

	err := s.db.Create(&cmt).Error
	s.NoError(err)
	return cmt.ID
}

func (s *HandlerTestSuite) TestCreateComment() {
	t := s.T()

	handlerFunc := func(t *testing.T, ctrl *gomock.Controller) *web.Handler {
		t.Helper()
		mockProducer := evtmocks.NewMockWechatRobotEventProducer(ctrl)
		mockProducer.EXPECT().Produce(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, event event.WechatRobotEvent) error {
			assert.NotEmpty(t, event.Robot)
			assert.NotEmpty(t, event.RawContent)
			return nil
		}).Times(1)
		svc := service.NewCommentService(nil, repository.NewCommentRepository(s.dao))
		return web.NewHandler(svc, mockProducer)
	}

	testCases := []struct {
		name           string
		newHandlerFunc func(t *testing.T, ctrl *gomock.Controller) *web.Handler
		reqFunc        func() web.CreateRequest
		wantCode       int
		wantErr        bool
		after          func(commentID int64)
	}{
		{
			name:           "创建始祖评论成功",
			newHandlerFunc: handlerFunc,
			reqFunc: func() web.CreateRequest {
				return web.CreateRequest{
					Comment: web.Comment{
						Biz:      "article",
						BizID:    s.getUniqueBizID(),
						ParentID: 0,
						Content:  "这是一个测试评论",
					},
				}
			},
			wantCode: 200,
			after: func(commentID int64) {
				var cmt dao.Comment
				err := s.db.First(&cmt, commentID).Error
				s.NoError(err)
				s.Equal("这是一个测试评论", cmt.Content)
				s.Equal(int64(0), cmt.ParentID.V)
				s.Equal(int64(0), cmt.AncestorID.V)
			},
		},
		{
			name: "创建始祖评论成功_发送消息失败",
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) *web.Handler {
				t.Helper()
				mockProducer := evtmocks.NewMockWechatRobotEventProducer(ctrl)
				mockProducer.EXPECT().Produce(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, event event.WechatRobotEvent) error {
					return errors.New("fake error")
				}).Times(1)
				svc := service.NewCommentService(nil, repository.NewCommentRepository(s.dao))
				return web.NewHandler(svc, mockProducer)

			},
			reqFunc: func() web.CreateRequest {
				return web.CreateRequest{
					Comment: web.Comment{
						Biz:      "article",
						BizID:    s.getUniqueBizID(),
						ParentID: 0,
						Content:  "这是一个测试发送消息失败评论",
					},
				}
			},
			wantCode: 200,
			after: func(commentID int64) {
				var cmt dao.Comment
				err := s.db.First(&cmt, commentID).Error
				s.NoError(err)
				s.Equal("这是一个测试发送消息失败评论", cmt.Content)
				s.Equal(int64(0), cmt.ParentID.V)
				s.Equal(int64(0), cmt.AncestorID.V)
			},
		},
		{
			name:           "创建一级回复成功",
			newHandlerFunc: handlerFunc,
			reqFunc: func() web.CreateRequest {
				biz := "article"
				bizID := s.getUniqueBizID()
				parentID := s.createAncestorComment(biz, bizID)
				return web.CreateRequest{
					Comment: web.Comment{
						Biz:      biz,
						BizID:    bizID,
						ParentID: parentID,
						Content:  "这是一个回复",
					},
				}
			},
			wantCode: 200,
			after: func(commentID int64) {
				var cmt dao.Comment
				err := s.db.First(&cmt, commentID).Error
				s.NoError(err)
				s.NotEqual(int64(0), cmt.ParentID.V)
				s.NotEqual(int64(0), cmt.AncestorID.V)
			},
		},
		{
			name:           "创建二级回复成功",
			newHandlerFunc: handlerFunc,
			reqFunc: func() web.CreateRequest {
				biz := "course"
				bizID := s.getUniqueBizID()
				parentID := s.createAncestorComment(biz, bizID)
				return web.CreateRequest{
					Comment: web.Comment{
						Biz:      biz,
						BizID:    bizID,
						ParentID: parentID,
						Content:  "这是二级回复",
					},
				}
			},
			wantCode: 200,
			after: func(commentID int64) {
				var cmt dao.Comment
				err := s.db.First(&cmt, commentID).Error
				s.NoError(err)
				s.NotEqual(int64(0), cmt.AncestorID.V)
			},
		},
		{
			name:           "创建多级深度回复成功",
			newHandlerFunc: handlerFunc,
			reqFunc: func() web.CreateRequest {
				biz := "course"
				bizID := s.getUniqueBizID()
				ancestorID := s.createAncestorComment(biz, bizID)
				reply1ID := s.createReplyComment(ancestorID, ancestorID, "一级回复")
				reply2ID := s.createReplyComment(reply1ID, ancestorID, "二级回复")
				return web.CreateRequest{
					Comment: web.Comment{
						Biz:      biz,
						BizID:    bizID,
						ParentID: reply2ID,
						Content:  "这是三级回复",
					},
				}
			},
			wantCode: 200,
			after: func(commentID int64) {
				var cmt dao.Comment
				err := s.db.First(&cmt, commentID).Error
				s.NoError(err)
				s.NotEqual(int64(0), cmt.AncestorID.V)
			},
		},
		{
			name:           "无效ParentID回复失败",
			newHandlerFunc: s.newHandlerWithout3rdDependency,
			reqFunc: func() web.CreateRequest {
				return web.CreateRequest{
					Comment: web.Comment{
						Biz:      "article",
						BizID:    s.getUniqueBizID(),
						ParentID: 99999999, // 不存在的ParentID
						Content:  "回复不存在的评论",
					},
				}
			},
			wantCode: 500,
			wantErr:  true,
		},
		{
			name:           "空内容评论失败",
			newHandlerFunc: s.newHandlerWithout3rdDependency,
			reqFunc: func() web.CreateRequest {
				return web.CreateRequest{
					Comment: web.Comment{
						Biz:      "article",
						BizID:    s.getUniqueBizID(),
						ParentID: 0,
						Content:  "",
					},
				}
			},
			wantCode: 500,
			wantErr:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			req := tc.reqFunc()

			httpReq, err := http.NewRequest(http.MethodPost,
				"/comment/", iox.NewJSONReader(req))
			s.NoError(err)
			httpReq.Header.Set("Content-Type", "application/json")
			recorder := test.NewJSONResponseRecorder[int64]()
			server := s.newGinServer(tc.newHandlerFunc(t, ctrl), testUID)
			server.ServeHTTP(recorder, httpReq)

			s.Equal(tc.wantCode, recorder.Code,
				fmt.Sprintf("Expected %d but got %d, Response: %s", tc.wantCode, recorder.Code, recorder.Body.String()))

			if !tc.wantErr && recorder.Code == 200 {
				if tc.after != nil {
					tc.after(recorder.MustScan().Data)
				}
			}
		})
	}
}

func (s *HandlerTestSuite) newHandlerWithout3rdDependency(t *testing.T, _ *gomock.Controller) *web.Handler {
	t.Helper()
	svc := service.NewCommentService(nil, repository.NewCommentRepository(s.dao))
	return web.NewHandler(svc, nil)
}

func (s *HandlerTestSuite) TestCommentList() {
	t := s.T()

	testCases := []struct {
		name           string
		newHandlerFunc func(t *testing.T, ctrl *gomock.Controller) *web.Handler
		reqFunc        func() web.ListRequest
		wantCode       int
		after          func(result web.CommentList)
	}{
		{
			name:           "始祖评论分页查询按时间倒序",
			newHandlerFunc: s.newHandlerWithMockUserServiceOnly,
			reqFunc: func() web.ListRequest {
				biz := "article"
				bizID := s.getUniqueBizID()

				ids := make([]int64, 3)
				for i := 0; i < 3; i++ {
					ids[i] = s.createAncestorComment(biz, bizID)
					time.Sleep(time.Millisecond)
				}
				return web.ListRequest{
					Biz:   biz,
					BizID: bizID,
					Limit: 10,
				}
			},
			wantCode: 200,
			after: func(result web.CommentList) {
				s.Equal(3, len(result.List))
				s.Equal(3, result.Total)
				if len(result.List) >= 2 {
					s.True(result.List[0].Utime >= result.List[1].Utime) // 倒序
				}
			},
		},
		{
			name:           "带子评论总数的预加载的分页查询",
			newHandlerFunc: s.newHandlerWithMockUserServiceOnly,
			reqFunc: func() web.ListRequest {
				biz := "article"
				bizID := s.getUniqueBizID()

				ancestorID := s.createAncestorComment(biz, bizID)
				ancestorID2 := s.createAncestorComment(biz, bizID)

				for i := range 5 {
					s.createReplyComment(ancestorID, ancestorID, fmt.Sprintf("子评论%d", i+1))
				}

				for i := range 2 {
					s.createReplyComment(ancestorID2, ancestorID2, fmt.Sprintf("子评论%d", i+1))
				}

				return web.ListRequest{
					Biz:   biz,
					BizID: bizID,
					Limit: 10,
				}
			},
			wantCode: 200,
			after: func(result web.CommentList) {
				s.Equal(2, len(result.List))
				for i := range result.List {
					if i == 0 {
						s.Equal(int64(2), result.List[i].ReplyCount)
					} else {
						s.Equal(int64(5), result.List[i].ReplyCount)
					}
					// 验证评论用户信息填充
					s.Equal("测试用户1", result.List[i].User.Nickname)
					s.Equal("avatar1.jpg", result.List[i].User.Avatar)
				}

			},
		},
		{
			name:           "边界分页测试",
			newHandlerFunc: s.newHandlerWithMockUserServiceOnly,
			reqFunc: func() web.ListRequest {
				biz := "article"
				bizID := s.getUniqueBizID()

				ids := make([]int64, 5)
				for i := 0; i < 5; i++ {
					ids[i] = s.createAncestorComment(biz, bizID)
					time.Sleep(time.Millisecond)
				}
				return web.ListRequest{
					Biz:   biz,
					BizID: bizID,
					Limit: 3,
				}
			},
			wantCode: 200,
			after: func(result web.CommentList) {
				s.Equal(3, len(result.List)) // 分页限制
				s.Equal(5, result.Total)     // 总数
			},
		},
		{
			name:           "查询不存在的业务资源",
			newHandlerFunc: s.newHandlerWithMockUserServiceOnly,
			reqFunc: func() web.ListRequest {
				return web.ListRequest{
					Biz:   "article",
					BizID: s.getUniqueBizID(),
					Limit: 10,
				}
			},
			wantCode: 200,
			after: func(result web.CommentList) {
				s.Equal(0, len(result.List))
				s.Equal(0, result.Total)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			httpReq, err := http.NewRequest(http.MethodPost,
				"/comment/list", iox.NewJSONReader(tc.reqFunc()))
			s.NoError(err)
			httpReq.Header.Set("Content-Type", "application/json")
			recorder := test.NewJSONResponseRecorder[web.CommentList]()

			server := s.newGinServer(tc.newHandlerFunc(t, ctrl), testUID)
			server.ServeHTTP(recorder, httpReq)

			s.Equal(tc.wantCode, recorder.Code)

			if recorder.Code == 200 {
				if tc.after != nil {
					tc.after(recorder.MustScan().Data)
				}
			}
		})
	}
}

func (s *HandlerTestSuite) newHandlerWithMockUserServiceOnly(t *testing.T, ctrl *gomock.Controller) *web.Handler {
	t.Helper()
	mockUserSvc := usermocks.NewMockUserService(ctrl)
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
				if u, exists := testUsers[id]; exists {
					users = append(users, u)
				}
			}
			return users, nil
		}).AnyTimes()
	svc := service.NewCommentService(mockUserSvc, repository.NewCommentRepository(s.dao))
	return web.NewHandler(svc, nil)
}

func (s *HandlerTestSuite) TestGetReplies() {
	t := s.T()

	testCases := []struct {
		name           string
		newHandlerFunc func(t *testing.T, ctrl *gomock.Controller) *web.Handler
		reqFunc        func() web.RepliesRequest
		wantCode       int
		after          func(result web.CommentList)
	}{
		{
			name:           "回复分页查询按时间倒序",
			newHandlerFunc: s.newHandlerWithMockUserServiceOnly,
			reqFunc: func() web.RepliesRequest {
				biz := "article"
				bizID := s.getUniqueBizID()
				ancestorID := s.createAncestorComment(biz, bizID)
				for i := 0; i < 2; i++ {
					reply1ID := s.createReplyComment(ancestorID, ancestorID, fmt.Sprintf("一级回复%d", i+1))
					for j := 0; j < 2; j++ {
						_ = s.createReplyComment(reply1ID, ancestorID, fmt.Sprintf("二级回复%d-%d", i+1, j+1))
					}
				}
				return web.RepliesRequest{
					AncestorID: ancestorID,
					MinID:      0,
					Limit:      10,
				}
			},
			wantCode: 200,
			after: func(result web.CommentList) {
				s.Equal(6, len(result.List)) // 2个一级 + 4个二级
				s.Equal(6, result.Total)

				// 验证按时间倒序排列
				if len(result.List) >= 2 {
					s.True(result.List[0].Utime >= result.List[1].Utime)
				}

				// 验证回复用户信息填充
				if len(result.List) > 0 {
					s.Equal("测试用户2", result.List[0].User.Nickname)
					s.Equal("avatar2.jpg", result.List[0].User.Avatar)
				}
			},
		},
		{
			name:           "分页查询回复",
			newHandlerFunc: s.newHandlerWithMockUserServiceOnly,
			reqFunc: func() web.RepliesRequest {
				biz := "article"
				bizID := s.getUniqueBizID()
				ancestorID := s.createAncestorComment(biz, bizID)
				for i := 0; i < 5; i++ {
					_ = s.createReplyComment(ancestorID, ancestorID, fmt.Sprintf("回复%d", i+1))
					time.Sleep(time.Millisecond)
				}

				return web.RepliesRequest{
					AncestorID: ancestorID,
					MinID:      0,
					Limit:      3,
				}
			},
			wantCode: 200,
			after: func(result web.CommentList) {
				s.Equal(3, len(result.List)) // 分页限制
				s.Equal(5, result.Total)     // 总数
			},
		},
		{
			name:           "查询不存在的始祖评论",
			newHandlerFunc: s.newHandlerWithMockUserServiceOnly,
			reqFunc: func() web.RepliesRequest {
				return web.RepliesRequest{
					AncestorID: 99999999,
					MinID:      0,
					Limit:      10,
				}
			},
			wantCode: 200,
			after: func(result web.CommentList) {
				s.Equal(0, len(result.List))
				s.Equal(0, result.Total)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			httpReq, err := http.NewRequest(http.MethodPost,
				"/comment/replies", iox.NewJSONReader(tc.reqFunc()))
			s.NoError(err)
			httpReq.Header.Set("Content-Type", "application/json")
			recorder := test.NewJSONResponseRecorder[web.CommentList]()

			server := s.newGinServer(tc.newHandlerFunc(t, ctrl), testUID)
			server.ServeHTTP(recorder, httpReq)

			s.Equal(tc.wantCode, recorder.Code)

			if recorder.Code == 200 {
				if tc.after != nil {
					tc.after(recorder.MustScan().Data)
				}
			}
		})
	}
}

func (s *HandlerTestSuite) TestDelete() {
	t := s.T()

	testCases := []struct {
		name           string
		newHandlerFunc func(t *testing.T, ctrl *gomock.Controller) *web.Handler
		before         func() (id int64)
		req            web.DeleteRequest
		wantCode       int
		wantResp       test.Result[any]
		after          func(id int64)
	}{
		{
			name:           "删除成功_始祖评论_无后代",
			newHandlerFunc: s.newHandlerWithout3rdDependency,
			before: func() (id int64) {
				return s.createAncestorComment("audio", s.getUniqueBizID())
			},
			req: web.DeleteRequest{
				ID: 0,
			},
			wantCode: 200,
			wantResp: test.Result[any]{
				Msg: "OK",
			},
			after: func(id int64) {
				_, err := s.dao.FindByID(context.Background(), id)
				s.Error(err)
			},
		},
		{
			name:           "删除成功_始祖评论_有后代",
			newHandlerFunc: s.newHandlerWithout3rdDependency,
			before: func() (id int64) {
				ancestorID := s.createAncestorComment("audio", s.getUniqueBizID())
				reply1ID := s.createReplyComment(ancestorID, ancestorID, "一级回复1")
				reply2ID := s.createReplyComment(ancestorID, ancestorID, "一级回复2")
				_ = s.createReplyComment(ancestorID, ancestorID, "一级回复3")
				_ = s.createReplyComment(reply1ID, ancestorID, "二级回复1")
				_ = s.createReplyComment(reply1ID, ancestorID, "二级回复2")
				_ = s.createReplyComment(reply1ID, ancestorID, "二级回复3")
				_ = s.createReplyComment(reply2ID, ancestorID, "二级回复1")
				return ancestorID
			},
			req: web.DeleteRequest{
				ID: 0,
			},
			wantCode: 200,
			wantResp: test.Result[any]{
				Msg: "OK",
			},
			after: func(id int64) {
				_, err := s.dao.FindByID(context.Background(), id)
				s.Error(err)

				descendants, err := s.dao.FindDescendants(context.Background(), id, math.MaxInt64, 100)
				s.NoError(err)
				s.Empty(descendants)
			},
		},
		{
			name:           "删除失败_评论ID不存在",
			newHandlerFunc: s.newHandlerWithout3rdDependency,
			before: func() (id int64) {
				return -1
			},
			req: web.DeleteRequest{
				ID: 0,
			},
			wantCode: 200,
			wantResp: test.Result[any]{
				Msg: "OK",
			},
			after: func(id int64) {},
		},
		{
			name:           "删除失败_操作者不是评论创建者",
			newHandlerFunc: s.newHandlerWithout3rdDependency,
			before: func() (id int64) {
				cmt := dao.Comment{
					Uid:        testUID3 + 101,
					Biz:        "audio",
					BizID:      s.getUniqueBizID(),
					ParentID:   sql.Null[int64]{V: 0, Valid: false},
					AncestorID: sql.Null[int64]{V: 0, Valid: false},
					Content:    fmt.Sprintf("始祖评论_%d", time.Now().UnixNano()),
					Ctime:      time.Now().UnixMilli(),
					Utime:      time.Now().UnixMilli(),
				}
				err := s.db.Create(&cmt).Error
				s.NoError(err)
				return cmt.ID
			},
			req: web.DeleteRequest{
				ID: 0,
			},
			wantCode: 200,
			wantResp: test.Result[any]{
				Msg: "OK",
			},
			after: func(id int64) {
				found, err := s.dao.FindByID(context.Background(), id)
				s.NoError(err)
				s.Equal(found.Uid, testUID3+101)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			id := tc.before()

			tc.req.ID = id

			httpReq, err := http.NewRequest(http.MethodPost,
				"/comment/delete", iox.NewJSONReader(tc.req))
			s.NoError(err)
			httpReq.Header.Set("Content-Type", "application/json")
			recorder := test.NewJSONResponseRecorder[any]()

			server := s.newGinServer(tc.newHandlerFunc(t, ctrl), testUID)
			server.ServeHTTP(recorder, httpReq)

			s.Equal(tc.wantCode, recorder.Code)
			s.Equal(tc.wantResp, recorder.MustScan())

			tc.after(id)
		})
	}
}

func TestHandler(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}
