package integration

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/ai/internal/domain"
	"github.com/ecodeclub/webook/internal/ai/internal/integration/startup"
	"github.com/ecodeclub/webook/internal/ai/internal/repository"
	"github.com/ecodeclub/webook/internal/ai/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/ai/internal/service"
	"github.com/ecodeclub/webook/internal/ai/internal/web"
	"github.com/ecodeclub/webook/internal/credit"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/econf"
	"github.com/gotomicro/ego/core/elog"
	"github.com/gotomicro/ego/server/egin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap/zapcore"
	"gorm.io/gorm"
)

func TestMockInterview(t *testing.T) {
	suite.Run(t, new(MockInterviewTestSuite))
}

type MockInterviewTestSuite struct {
	suite.Suite
	db               *gorm.DB
	mockInterviewHdl *web.MockInterviewHandler
	mockInterviewSvc service.MockInterviewService
	server           *egin.Component
}

func (s *MockInterviewTestSuite) SetupSuite() {
	db := testioc.InitDB()
	s.db = db
	err := dao.InitTables(db)
	s.NoError(err)
	s.mockInterviewSvc = service.NewMockInterviewService(repository.NewMockInterviewRepository(dao.NewMockInterviewDAO(s.db)))

	// 先插入 BizConfig
	mou, err := startup.InitModule(s.db, nil, nil, nil, &credit.Module{}, nil)
	s.NoError(err)
	s.mockInterviewHdl = mou.MockInterviewHdl

	econf.Set("server", map[string]any{"contextTimeout": "10m"})
	server := egin.Load("server").Build()

	// 添加 CORS 配置（与 gin.go 中的配置保持一致）
	server.Use(cors.New(cors.Config{
		ExposeHeaders:    []string{"X-Refresh-Token", "X-Access-Token"},
		AllowCredentials: true,
		AllowHeaders: []string{"X-Timestamp",
			"X-APP",
			"Authorization", "Content-Type"},
		AllowOriginFunc: func(origin string) bool {
			if strings.HasPrefix(origin, "http://localhost") {
				return true
			}
			// 允许本地开发环境
			return strings.Contains(origin, "localhost:63342") ||
				strings.Contains(origin, "127.0.0.1")
		},
	}))

	// 添加 session 模拟
	server.Use(func(ctx *gin.Context) {
		ctx.Set("_session", session.NewMemorySession(session.Claims{
			Uid: 123,
		}))
	})

	s.mockInterviewHdl.PrivateRoutes(server.Engine)
	s.server = server
}

func (s *MockInterviewTestSuite) SetupTest() {
	// 清理测试数据
	err := s.db.Exec("TRUNCATE TABLE `mock_interviews`").Error
	s.NoError(err)
	err = s.db.Exec("TRUNCATE TABLE `mock_interview_questions`").Error
	s.NoError(err)
}

func (s *MockInterviewTestSuite) TestE2E() {
	elog.DefaultLogger.SetLevel(zapcore.DebugLevel)
	t := s.T()

	cosSecretID := os.Getenv("COS_SECRET_ID")
	if cosSecretID == "" {
		t.Skip("未设置 COS_SECRET_ID 环境变量")
	}
	econf.Set("cos.secretID", cosSecretID)

	cosSecretKey := os.Getenv("COS_SECRET_KEY")
	if cosSecretKey == "" {
		t.Skip("未设置 COS_SECRET_KEY 环境变量")
	}
	econf.Set("cos.secretKey", cosSecretKey)

	cosBucket := os.Getenv("COS_BUCKET_NAME")
	if cosBucket == "" {
		t.Skip("未设置 COS_BUCKET_NAME 环境变量（格式：bucketname-appid）")
	}
	econf.Set("cos.bucket", cosBucket)

	cosRegion := os.Getenv("COS_REGION")
	if cosRegion == "" {
		t.Skip("未设置 COS_REGION 环境变量（如：ap-guangzhou）")
	}
	econf.Set("cos.region", cosRegion)

	if err := s.server.Run("localhost:8080"); err != nil {
		t.Skip(err)
	}
}

func (s *MockInterviewTestSuite) TestService_SaveInterview() {
	t := s.T()
	sn := fmt.Sprintf("sn-%d", time.Now().UnixNano())
	id, err := s.mockInterviewSvc.SaveInterview(t.Context(), domain.MockInterview{
		Uid:    123,
		Title:  "面试-保存用例",
		ChatSN: sn,
	})
	require.NoError(t, err)
	require.Greater(t, id, int64(0))

	t.Cleanup(func() {
		_ = s.db.Where("id IN ?", id).Delete(&dao.MockInterview{}).Error
	})

	// 列表：仅查自己的
	listMine, totalMine, err := s.mockInterviewSvc.ListInterviews(t.Context(), 123, 10, 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(listMine), 1)
	assert.GreaterOrEqual(t, totalMine, int64(1))
	// 校验列表中包含刚创建的这条
	found := false
	for _, it := range listMine {
		if it.ID == id {
			found = true
			assert.Equal(t, sn, it.ChatSN)
			assert.Equal(t, int64(123), it.Uid)
			assert.Equal(t, "面试-保存用例", it.Title)
			break
		}
	}
	assert.True(t, found, "ListInterviews 未包含刚创建的面试记录")
}

func (s *MockInterviewTestSuite) TestService_ListInterviews() {
	t := s.T()
	type payload struct {
		mineIDs  []int64
		otherIDs []int64
	}
	cases := []struct {
		name   string
		uid    int64
		limit  int
		offset int
		before func(t *testing.T, totalN int) payload
		after  func(t *testing.T, got []domain.MockInterview, total int64, p payload)
	}{
		{
			name:   "用户: 小于一页",
			uid:    123,
			limit:  5,
			offset: 0,
			before: func(t *testing.T, totalN int) payload {
				sn1 := fmt.Sprintf("sn-%d", time.Now().UnixNano())
				id1, err := s.mockInterviewSvc.SaveInterview(t.Context(), domain.MockInterview{Uid: 123, Title: "面试-A", ChatSN: sn1})
				require.NoError(t, err)
				sn2 := fmt.Sprintf("sn-%d", time.Now().UnixNano())
				id2, err := s.mockInterviewSvc.SaveInterview(t.Context(), domain.MockInterview{Uid: 123, Title: "面试-B", ChatSN: sn2})
				require.NoError(t, err)
				snOther := fmt.Sprintf("sn-%d", time.Now().UnixNano())
				idOther, err := s.mockInterviewSvc.SaveInterview(t.Context(), domain.MockInterview{Uid: 456, Title: "面试-C", ChatSN: snOther})
				require.NoError(t, err)
				ids := []int64{id1, id2, idOther}
				t.Cleanup(func() {
					_ = s.db.Where("id IN ?", ids).Delete(&dao.MockInterview{}).Error
				})
				return payload{mineIDs: []int64{id1, id2}, otherIDs: []int64{idOther}}
			},
			after: func(t *testing.T, got []domain.MockInterview, total int64, p payload) {
				assert.Equal(t, 2, len(got))
				assert.Equal(t, int64(2), total)
				have := map[int64]struct{}{}
				for _, it := range got {
					assert.Equal(t, int64(123), it.Uid)
					have[it.ID] = struct{}{}
				}
				for _, id := range p.mineIDs {
					_, ok := have[id]
					assert.True(t, ok)
				}
				for _, id := range p.otherIDs {
					_, ok := have[id]
					assert.False(t, ok)
				}
			},
		},
		{
			name:   "用户: 一页刚好",
			uid:    123,
			limit:  5,
			offset: 0,
			before: func(t *testing.T, totalN int) payload {
				sn1 := fmt.Sprintf("sn-%d", time.Now().UnixNano())
				id1, err := s.mockInterviewSvc.SaveInterview(t.Context(), domain.MockInterview{Uid: 123, Title: "面试-A", ChatSN: sn1})
				require.NoError(t, err)
				sn2 := fmt.Sprintf("sn-%d", time.Now().UnixNano())
				id2, err := s.mockInterviewSvc.SaveInterview(t.Context(), domain.MockInterview{Uid: 123, Title: "面试-B", ChatSN: sn2})
				require.NoError(t, err)
				snOther := fmt.Sprintf("sn-%d", time.Now().UnixNano())
				idOther, err := s.mockInterviewSvc.SaveInterview(t.Context(), domain.MockInterview{Uid: 456, Title: "面试-C", ChatSN: snOther})
				require.NoError(t, err)
				ids := []int64{id1, id2, idOther}
				t.Cleanup(func() {
					_ = s.db.Where("id IN ?", ids).Delete(&dao.MockInterview{}).Error
				})
				return payload{mineIDs: []int64{id1, id2}, otherIDs: []int64{idOther}}
			},
			after: func(t *testing.T, got []domain.MockInterview, total int64, p payload) {
				assert.Equal(t, 2, len(got))
				assert.Equal(t, int64(2), total)
				have := map[int64]struct{}{}
				for _, it := range got {
					assert.Equal(t, int64(123), it.Uid)
					have[it.ID] = struct{}{}
				}
				for _, id := range p.mineIDs {
					_, ok := have[id]
					assert.True(t, ok)
				}
				for _, id := range p.otherIDs {
					_, ok := have[id]
					assert.False(t, ok)
				}
			},
		},
		{
			name:   "用户: 两页不满",
			uid:    123,
			limit:  5,
			offset: 0,
			before: func(t *testing.T, totalN int) payload {
				sn1 := fmt.Sprintf("sn-%d", time.Now().UnixNano())
				id1, err := s.mockInterviewSvc.SaveInterview(t.Context(), domain.MockInterview{Uid: 123, Title: "面试-A", ChatSN: sn1})
				require.NoError(t, err)
				sn2 := fmt.Sprintf("sn-%d", time.Now().UnixNano())
				id2, err := s.mockInterviewSvc.SaveInterview(t.Context(), domain.MockInterview{Uid: 123, Title: "面试-B", ChatSN: sn2})
				require.NoError(t, err)
				snOther := fmt.Sprintf("sn-%d", time.Now().UnixNano())
				idOther, err := s.mockInterviewSvc.SaveInterview(t.Context(), domain.MockInterview{Uid: 456, Title: "面试-C", ChatSN: snOther})
				require.NoError(t, err)
				ids := []int64{id1, id2, idOther}
				t.Cleanup(func() {
					_ = s.db.Where("id IN ?", ids).Delete(&dao.MockInterview{}).Error
				})
				return payload{mineIDs: []int64{id1, id2}, otherIDs: []int64{idOther}}
			},
			after: func(t *testing.T, got []domain.MockInterview, total int64, p payload) {
				assert.Equal(t, 2, len(got))
				assert.Equal(t, int64(2), total)
				have := map[int64]struct{}{}
				for _, it := range got {
					assert.Equal(t, int64(123), it.Uid)
					have[it.ID] = struct{}{}
				}
				for _, id := range p.mineIDs {
					_, ok := have[id]
					assert.True(t, ok)
				}
				for _, id := range p.otherIDs {
					_, ok := have[id]
					assert.False(t, ok)
				}
			},
		},
		{
			name:   "用户: 两页刚好",
			uid:    123,
			limit:  5,
			offset: 0,
			before: func(t *testing.T, totalN int) payload {
				sn1 := fmt.Sprintf("sn-%d", time.Now().UnixNano())
				id1, err := s.mockInterviewSvc.SaveInterview(t.Context(), domain.MockInterview{Uid: 123, Title: "面试-A", ChatSN: sn1})
				require.NoError(t, err)
				sn2 := fmt.Sprintf("sn-%d", time.Now().UnixNano())
				id2, err := s.mockInterviewSvc.SaveInterview(t.Context(), domain.MockInterview{Uid: 123, Title: "面试-B", ChatSN: sn2})
				require.NoError(t, err)
				snOther := fmt.Sprintf("sn-%d", time.Now().UnixNano())
				idOther, err := s.mockInterviewSvc.SaveInterview(t.Context(), domain.MockInterview{Uid: 456, Title: "面试-C", ChatSN: snOther})
				require.NoError(t, err)
				ids := []int64{id1, id2, idOther}
				t.Cleanup(func() {
					_ = s.db.Where("id IN ?", ids).Delete(&dao.MockInterview{}).Error
				})
				return payload{mineIDs: []int64{id1, id2}, otherIDs: []int64{idOther}}
			},
			after: func(t *testing.T, got []domain.MockInterview, total int64, p payload) {
				assert.Equal(t, 2, len(got))
				assert.Equal(t, int64(2), total)
				have := map[int64]struct{}{}
				for _, it := range got {
					assert.Equal(t, int64(123), it.Uid)
					have[it.ID] = struct{}{}
				}
				for _, id := range p.mineIDs {
					_, ok := have[id]
					assert.True(t, ok)
				}
				for _, id := range p.otherIDs {
					_, ok := have[id]
					assert.False(t, ok)
				}
			},
		},
		{
			name:   "admin: 一页刚好",
			uid:    0,
			limit:  5,
			offset: 0,
			before: func(t *testing.T, totalN int) payload {
				sn1 := fmt.Sprintf("sn-%d", time.Now().UnixNano())
				id1, err := s.mockInterviewSvc.SaveInterview(t.Context(), domain.MockInterview{Uid: 123, Title: "面试-A", ChatSN: sn1})
				require.NoError(t, err)
				sn2 := fmt.Sprintf("sn-%d", time.Now().UnixNano())
				id2, err := s.mockInterviewSvc.SaveInterview(t.Context(), domain.MockInterview{Uid: 123, Title: "面试-B", ChatSN: sn2})
				require.NoError(t, err)
				snOther := fmt.Sprintf("sn-%d", time.Now().UnixNano())
				idOther, err := s.mockInterviewSvc.SaveInterview(t.Context(), domain.MockInterview{Uid: 456, Title: "面试-C", ChatSN: snOther})
				require.NoError(t, err)
				ids := []int64{id1, id2, idOther}
				t.Cleanup(func() {
					_ = s.db.Where("id IN ?", ids).Delete(&dao.MockInterview{}).Error
				})
				return payload{mineIDs: []int64{id1, id2}, otherIDs: []int64{idOther}}
			},
			after: func(t *testing.T, got []domain.MockInterview, total int64, p payload) {
				assert.Equal(t, 3, len(got))
				assert.Equal(t, int64(3), total)
				have := map[int64]struct{}{}
				for _, it := range got {
					have[it.ID] = struct{}{}
				}
				for _, id := range append(p.mineIDs, p.otherIDs...) {
					_, ok := have[id]
					assert.True(t, ok)
				}
			},
		},
		{
			name:   "admin: 两页刚好",
			uid:    0,
			limit:  5,
			offset: 0,
			before: func(t *testing.T, totalN int) payload {
				sn1 := fmt.Sprintf("sn-%d", time.Now().UnixNano())
				id1, err := s.mockInterviewSvc.SaveInterview(t.Context(), domain.MockInterview{Uid: 123, Title: "面试-A", ChatSN: sn1})
				require.NoError(t, err)
				sn2 := fmt.Sprintf("sn-%d", time.Now().UnixNano())
				id2, err := s.mockInterviewSvc.SaveInterview(t.Context(), domain.MockInterview{Uid: 123, Title: "面试-B", ChatSN: sn2})
				require.NoError(t, err)
				snOther := fmt.Sprintf("sn-%d", time.Now().UnixNano())
				idOther, err := s.mockInterviewSvc.SaveInterview(t.Context(), domain.MockInterview{Uid: 456, Title: "面试-C", ChatSN: snOther})
				require.NoError(t, err)
				ids := []int64{id1, id2, idOther}
				t.Cleanup(func() {
					_ = s.db.Where("id IN ?", ids).Delete(&dao.MockInterview{}).Error
				})
				return payload{mineIDs: []int64{id1, id2}, otherIDs: []int64{idOther}}
			},
			after: func(t *testing.T, got []domain.MockInterview, total int64, p payload) {
				assert.Equal(t, 3, len(got))
				assert.Equal(t, int64(3), total)
				have := map[int64]struct{}{}
				for _, it := range got {
					have[it.ID] = struct{}{}
				}
				for _, id := range append(p.mineIDs, p.otherIDs...) {
					_, ok := have[id]
					assert.True(t, ok)
				}
			},
		},
	}

	for _, cs := range cases {
		t.Run(cs.name, func(t *testing.T) {
			totalN := 5 // 每个用例创建 5 条数据
			p := cs.before(t, totalN)
			list, total, err := s.mockInterviewSvc.ListInterviews(t.Context(), cs.uid, cs.limit, cs.offset)
			require.NoError(t, err)
			cs.after(t, list, total, p)
		})
	}
}

func (s *MockInterviewTestSuite) TestService_SaveQuestion() {
	t := s.T()
	// 先保存一场面试
	sn := fmt.Sprintf("sn-%d", time.Now().UnixNano())
	id, err := s.mockInterviewSvc.SaveInterview(t.Context(), domain.MockInterview{
		Uid:    123,
		Title:  "面试-题目用例",
		ChatSN: sn,
	})
	require.NoError(t, err)
	require.Greater(t, id, int64(0))

	t.Cleanup(func() {
		_ = s.db.Where("id IN ?", id).Delete(&dao.MockInterview{}).Error
	})

	// 记录一道“自由生成”的题目（generated*）
	qid, err := s.mockInterviewSvc.SaveQuestion(t.Context(), domain.MockInterviewQuestion{
		ChatSN: sn,
		Uid:    123,
		Biz:    "generated",
		BizID:  0,
		Title:  "自拟题-一号",
		Answer: map[string]any{"text": "我的回答"},
	})
	require.NoError(t, err)
	require.Greater(t, qid, int64(0))

	t.Cleanup(func() {
		_ = s.db.Where("id IN ?", qid).Delete(&dao.MockInterviewQuestion{}).Error
	})

	// 列表查询
	qs, totalQ, err := s.mockInterviewSvc.ListQuestions(t.Context(), id, 123, 10, 0)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(qs), 1)
	assert.GreaterOrEqual(t, totalQ, int64(1))
	found := false
	for _, q := range qs {
		if q.ID == qid {
			found = true
			assert.Equal(t, id, q.InterviewID)
			assert.Equal(t, sn, q.ChatSN)
			assert.Equal(t, "generated", q.Biz)
			assert.Equal(t, int64(0), q.BizID)
			assert.Equal(t, "自拟题-一号", q.Title)
		}
	}
	assert.True(t, found, "未在列表中找到刚插入的题目")
}

func (s *MockInterviewTestSuite) TestService_SaveQuestion_Failed() {
	t := s.T()
	// 先准备一场面试，拿到 chat_sn
	sn := fmt.Sprintf("sn-%d", time.Now().UnixNano())
	id, err := s.mockInterviewSvc.SaveInterview(t.Context(), domain.MockInterview{
		Uid:    123,
		Title:  "面试-题目失败用例",
		ChatSN: sn,
	})
	require.NoError(t, err)
	require.Greater(t, id, int64(0))

	t.Cleanup(func() {
		_ = s.db.Where("id IN ?", id).Delete(&dao.MockInterview{}).Error
	})

	cases := []struct {
		name string
		q    domain.MockInterviewQuestion
	}{
		{
			name: "biz 为空",
			q:    domain.MockInterviewQuestion{ChatSN: sn, Uid: 123, Biz: "", BizID: 0, Title: "任意"},
		},
		{
			name: "generated 无标题",
			q:    domain.MockInterviewQuestion{ChatSN: sn, Uid: 123, Biz: "generated", BizID: 0, Title: ""},
		},
		{
			name: "question bizID=0",
			q:    domain.MockInterviewQuestion{ChatSN: sn, Uid: 123, Biz: "question", BizID: 0, Title: ""},
		},
		{
			name: "question 标题非空",
			q:    domain.MockInterviewQuestion{ChatSN: sn, Uid: 123, Biz: "question", BizID: 1001, Title: "不应提供"},
		},
	}

	for _, cs := range cases {
		t.Run(cs.name, func(t *testing.T) {
			_, err := s.mockInterviewSvc.SaveQuestion(t.Context(), cs.q)
			assert.Error(t, err)
		})
	}
}

func (s *MockInterviewTestSuite) TestService_ListQuestions() {
	t := s.T()
	type payloadQ struct {
		interviewID int64
		mineIDs     []int64
		otherIDs    []int64
		sn          string
	}
	cases := []struct {
		name   string
		uid    int64
		limit  int
		offset int
		before func(t *testing.T, totalN int) payloadQ
		after  func(t *testing.T, got []domain.MockInterviewQuestion, total int64, p payloadQ)
	}{
		{
			name:   "仅查自己的题目(uid=123)",
			uid:    123,
			limit:  100,
			offset: 0,
			before: func(t *testing.T, totalN int) payloadQ {
				sn := fmt.Sprintf("sn-%d", time.Now().UnixNano())
				interviewID, err := s.mockInterviewSvc.SaveInterview(t.Context(), domain.MockInterview{Uid: 123, Title: "面试-列表题目", ChatSN: sn})
				require.NoError(t, err)
				qid1, err := s.mockInterviewSvc.SaveQuestion(t.Context(), domain.MockInterviewQuestion{ChatSN: sn, Uid: 123, Biz: "generated", BizID: 0, Title: "题目-1", Answer: map[string]any{"text": "答案-1"}})
				require.NoError(t, err)
				qid2, err := s.mockInterviewSvc.SaveQuestion(t.Context(), domain.MockInterviewQuestion{ChatSN: sn, Uid: 123, Biz: "question", BizID: 2001, Title: "", Answer: map[string]any{"text": "答案-2"}})
				require.NoError(t, err)
				// 额外插入他人题目
				qid3, err := s.mockInterviewSvc.SaveQuestion(t.Context(), domain.MockInterviewQuestion{ChatSN: sn, Uid: 456, Biz: "generated", BizID: 0, Title: "他人题", Answer: map[string]any{"text": "他人"}})
				require.NoError(t, err)
				qids := []int64{qid1, qid2, qid3}
				t.Cleanup(func() {
					_ = s.db.Where("id IN ?", qids).Delete(&dao.MockInterviewQuestion{}).Error
					_ = s.db.Where("id = ?", interviewID).Delete(&dao.MockInterview{}).Error
				})
				return payloadQ{interviewID: interviewID, mineIDs: []int64{qid1, qid2}, otherIDs: nil, sn: sn}
			},
			after: func(t *testing.T, got []domain.MockInterviewQuestion, total int64, p payloadQ) {
				assert.GreaterOrEqual(t, len(got), 2)
				assert.GreaterOrEqual(t, total, int64(2))
				have := map[int64]struct{}{}
				for _, it := range got {
					assert.Equal(t, int64(123), it.Uid)
					assert.Equal(t, p.interviewID, it.InterviewID)
					have[it.ID] = struct{}{}
				}
				for _, id := range p.mineIDs {
					_, ok := have[id]
					assert.True(t, ok)
				}
			},
		},
		{
			name:   "admin 查询全部题目(uid=0)",
			uid:    0,
			limit:  100,
			offset: 0,
			before: func(t *testing.T, totalN int) payloadQ {
				sn := fmt.Sprintf("sn-%d", time.Now().UnixNano())
				interviewID, err := s.mockInterviewSvc.SaveInterview(t.Context(), domain.MockInterview{Uid: 123, Title: "面试-列表题目-admin", ChatSN: sn})
				require.NoError(t, err)
				qid1, err := s.mockInterviewSvc.SaveQuestion(t.Context(), domain.MockInterviewQuestion{ChatSN: sn, Uid: 123, Biz: "generated", BizID: 0, Title: "题目-1", Answer: map[string]any{"text": "答案-1"}})
				require.NoError(t, err)
				// 他人题目
				qidOther, err := s.mockInterviewSvc.SaveQuestion(t.Context(), domain.MockInterviewQuestion{ChatSN: sn, Uid: 456, Biz: "question", BizID: 3001, Title: "", Answer: map[string]any{"text": "他人-题库"}})
				require.NoError(t, err)
				qids := []int64{qid1, qidOther}
				t.Cleanup(func() {
					_ = s.db.Where("id IN ?", qids).Delete(&dao.MockInterviewQuestion{}).Error
					_ = s.db.Where("id = ?", interviewID).Delete(&dao.MockInterview{}).Error
				})
				return payloadQ{interviewID: interviewID, mineIDs: []int64{qid1}, otherIDs: []int64{qidOther}, sn: sn}
			},
			after: func(t *testing.T, got []domain.MockInterviewQuestion, total int64, p payloadQ) {
				assert.GreaterOrEqual(t, len(got), 2)
				assert.GreaterOrEqual(t, total, int64(2))
				have := map[int64]struct{}{}
				for _, it := range got {
					have[it.ID] = struct{}{}
				}
				for _, id := range append(p.mineIDs, p.otherIDs...) {
					_, ok := have[id]
					assert.True(t, ok)
				}
			},
		},
	}

	for _, cs := range cases {
		t.Run(cs.name, func(t *testing.T) {
			totalN := 5 // 每个用例创建 5 条数据
			p := cs.before(t, totalN)
			list, total, err := s.mockInterviewSvc.ListQuestions(t.Context(), p.interviewID, cs.uid, cs.limit, cs.offset)
			require.NoError(t, err)
			cs.after(t, list, total, p)
		})
	}
}
