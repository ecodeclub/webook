//go:build e2e

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	streamhdlmocks "github.com/ecodeclub/webook/internal/ai/internal/service/llm/handler/stream_mocks"

	"github.com/ecodeclub/ekit/iox"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/ai/internal/web"
	"github.com/ecodeclub/webook/internal/test"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/econf"
	"github.com/gotomicro/ego/server/egin"

	"github.com/ecodeclub/ekit/sqlx"
	"github.com/ecodeclub/webook/internal/ai/internal/service/llm"
	hdlmocks "github.com/ecodeclub/webook/internal/ai/internal/service/llm/handler/mocks"

	"github.com/ecodeclub/webook/internal/ai/internal/domain"
	"github.com/ecodeclub/webook/internal/ai/internal/integration/startup"
	"github.com/ecodeclub/webook/internal/ai/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/credit"
	creditmocks "github.com/ecodeclub/webook/internal/credit/mocks"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/ego-component/egorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const knowledgeId = "abc"

type LLMServiceSuite struct {
	suite.Suite
	logDao dao.LLMRecordDAO
	db     *egorm.Component
	svc    llm.Service
}

func TestLLMServiceSuite(t *testing.T) {
	suite.Run(t, new(LLMServiceSuite))
}

func (s *LLMServiceSuite) SetupSuite() {
	db := testioc.InitDB()
	s.db = db
	err := dao.InitTables(db)
	s.NoError(err)
	s.logDao = dao.NewGORMLLMLogDAO(db)

	// 先插入 BizConfig
	now := time.Now().UnixMilli()
	err = s.db.Create(&dao.BizConfig{
		Biz:            domain.BizQuestionExamine,
		MaxInput:       100,
		Price:          1,
		PromptTemplate: "这是问题 %s，这是问题内容 %s，这是用户输入 %s",
		KnowledgeId:    knowledgeId,
		Ctime:          now,
		Utime:          now,
	}).Error
	s.NoError(err)
	err = s.db.Create(&dao.BizConfig{
		Biz:            domain.BizCaseExamine,
		MaxInput:       100,
		Price:          1,
		PromptTemplate: "这是案例 %s，这是案例内容 %s，这是用户输入 %s",
		KnowledgeId:    knowledgeId,
		Ctime:          now,
		Utime:          now,
	}).Error
	s.NoError(err)

	err = s.db.Create(&dao.BizConfig{
		Biz:            domain.AnalysisJDBiz,
		MaxInput:       100,
		PromptTemplate: "这是岗位描述 %s",
		KnowledgeId:    knowledgeId,
		Ctime:          now,
		Utime:          now,
	}).Error
	s.NoError(err)

	err = s.db.Create(&dao.BizConfig{
		Biz:            domain.AnalysisJDTech,
		MaxInput:       100,
		PromptTemplate: "这是岗位描述tech %s",
		KnowledgeId:    knowledgeId,
		Ctime:          now,
		Utime:          now,
	}).Error
	s.NoError(err)

	err = s.db.Create(&dao.BizConfig{
		Biz:            domain.AnalysisJDPosition,
		MaxInput:       100,
		PromptTemplate: "这是岗位描述position %s",
		KnowledgeId:    knowledgeId,
		Ctime:          now,
		Utime:          now,
	}).Error
	s.NoError(err)

	err = s.db.Create(&dao.BizConfig{
		Biz:            domain.AnalysisJDSubtext,
		MaxInput:       100,
		PromptTemplate: "这是岗位描述Subtext %s",
		KnowledgeId:    knowledgeId,
		Ctime:          now,
		Utime:          now,
	}).Error
	s.NoError(err)
}

func (s *LLMServiceSuite) TearDownSuite() {
	err := s.db.Exec("TRUNCATE TABLE `ai_biz_configs`").Error
	require.NoError(s.T(), err)
}

func (s *LLMServiceSuite) TearDownTest() {
	err := s.db.Exec("TRUNCATE TABLE `llm_records`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE `llm_credits`").Error
	require.NoError(s.T(), err)
}

func (s *LLMServiceSuite) TestService() {
	t := s.T()
	testCases := []struct {
		name       string
		req        domain.LLMRequest
		before     func(t *testing.T, ctrl *gomock.Controller) (*hdlmocks.MockHandler, credit.Service)
		assertFunc assert.ErrorAssertionFunc
		after      func(t *testing.T, resp domain.LLMResponse)
	}{
		{
			name: "八股文测试-成功",
			req: domain.LLMRequest{
				Biz: domain.BizQuestionExamine,
				Uid: 123,
				Tid: "11",
				Input: []string{
					"问题1",
					"问题1内容",
					"用户输入1",
				},
			},
			assertFunc: assert.NoError,
			before: func(t *testing.T,
				ctrl *gomock.Controller) (*hdlmocks.MockHandler, credit.Service) {
				llmHdl := hdlmocks.NewMockHandler(ctrl)
				llmHdl.EXPECT().Handle(gomock.Any(), gomock.Any()).
					Return(domain.LLMResponse{
						Tokens: 100,
						Amount: 100,
						Answer: "aians",
					}, nil)
				creditSvc := creditmocks.NewMockService(ctrl)
				creditSvc.EXPECT().GetCreditsByUID(gomock.Any(), gomock.Any()).Return(credit.Credit{
					TotalAmount: 1000,
				}, nil)
				creditSvc.EXPECT().TryDeductCredits(gomock.Any(), gomock.Any()).Return(11, nil)
				creditSvc.EXPECT().ConfirmDeductCredits(gomock.Any(), int64(123), int64(11)).Return(nil)
				return llmHdl, creditSvc
			},
			after: func(t *testing.T, resp domain.LLMResponse) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
				defer cancel()
				// 校验response写入的内容是否正确
				assert.Equal(t, domain.LLMResponse{
					Tokens: 100,
					Amount: 100,
					Answer: "aians",
				}, resp)
				var logModel dao.LLMRecord
				err := s.db.WithContext(ctx).Where("id = ?", 1).First(&logModel).Error
				require.NoError(t, err)
				s.assertLog(dao.LLMRecord{
					Id:          1,
					Tid:         "11",
					Uid:         123,
					Biz:         domain.BizQuestionExamine,
					Tokens:      100,
					Amount:      100,
					KnowledgeId: knowledgeId,
					Input: sqlx.JsonColumn[[]string]{
						Valid: true,
						Val: []string{
							"问题1",
							"问题1内容",
							"用户输入1",
						},
					},
					Status:         1,
					PromptTemplate: sqlx.NewNullString("这是问题 %s，这是问题内容 %s，这是用户输入 %s"),
					Answer:         sqlx.NewNullString("aians"),
				}, logModel)
				// 校验credit写入的内容是否正确
				var creditLogModel dao.LLMCredit
				err = s.db.WithContext(ctx).Where("id = ?", 1).First(&creditLogModel).Error
				require.NoError(t, err)
				s.assertCreditLog(dao.LLMCredit{
					Id:     1,
					Tid:    "11",
					Uid:    123,
					Biz:    domain.BizQuestionExamine,
					Amount: 100,
					Status: 1,
				}, creditLogModel)
			},
		},
		{
			name: "案例测试-成功",
			req: domain.LLMRequest{
				Biz: domain.BizCaseExamine,
				Uid: 123,
				Tid: "13",
				Input: []string{
					"案例1",
					"案例1内容",
					"用户输入1",
				},
			},
			assertFunc: assert.NoError,
			before: func(t *testing.T,
				ctrl *gomock.Controller) (*hdlmocks.MockHandler, credit.Service) {
				llmHdl := hdlmocks.NewMockHandler(ctrl)
				llmHdl.EXPECT().Handle(gomock.Any(), gomock.Any()).
					Return(domain.LLMResponse{
						Tokens: 100,
						Amount: 100,
						Answer: "aians",
					}, nil)
				creditSvc := creditmocks.NewMockService(ctrl)
				creditSvc.EXPECT().GetCreditsByUID(gomock.Any(), gomock.Any()).Return(credit.Credit{
					TotalAmount: 1000,
				}, nil)
				creditSvc.EXPECT().TryDeductCredits(gomock.Any(), gomock.Any()).Return(11, nil)
				creditSvc.EXPECT().ConfirmDeductCredits(gomock.Any(), int64(123), int64(11)).Return(nil)
				return llmHdl, creditSvc
			},
			after: func(t *testing.T, resp domain.LLMResponse) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
				defer cancel()
				// 校验response写入的内容是否正确
				assert.Equal(t, domain.LLMResponse{
					Tokens: 100,
					Amount: 100,
					Answer: "aians",
				}, resp)
				var logModel dao.LLMRecord
				err := s.db.WithContext(ctx).Where("tid = ?", "13").First(&logModel).Error
				require.NoError(t, err)
				logModel.Id = 0
				s.assertLog(dao.LLMRecord{
					Id:          0,
					Tid:         "13",
					Uid:         123,
					Biz:         domain.BizCaseExamine,
					Tokens:      100,
					Amount:      100,
					KnowledgeId: knowledgeId,
					Input: sqlx.JsonColumn[[]string]{
						Valid: true,
						Val: []string{
							"案例1",
							"案例1内容",
							"用户输入1",
						},
					},
					Status:         1,
					PromptTemplate: sqlx.NewNullString("这是案例 %s，这是案例内容 %s，这是用户输入 %s"),
					Answer:         sqlx.NewNullString("aians"),
				}, logModel)
				// 校验credit写入的内容是否正确
				var creditLogModel dao.LLMCredit
				err = s.db.WithContext(ctx).Where("tid = ?", "13").First(&creditLogModel).Error
				require.NoError(t, err)
				assert.True(t, creditLogModel.Id != 0)
				creditLogModel.Id = 0
				s.assertCreditLog(dao.LLMCredit{
					Tid:    "13",
					Uid:    123,
					Biz:    domain.BizCaseExamine,
					Amount: 100,
					Status: 1,
				}, creditLogModel)
			},
		},
		{
			name: "积分不足",
			req: domain.LLMRequest{
				Biz: domain.BizQuestionExamine,
				Uid: 124,
				Tid: "11",
				Input: []string{
					"nihao",
				},
			},
			before: func(t *testing.T,
				ctrl *gomock.Controller) (*hdlmocks.MockHandler, credit.Service) {
				llmHdl := hdlmocks.NewMockHandler(ctrl)
				creditSvc := creditmocks.NewMockService(ctrl)
				creditSvc.EXPECT().GetCreditsByUID(gomock.Any(), gomock.Any()).Return(credit.Credit{
					TotalAmount: 0,
				}, nil)
				return llmHdl, creditSvc
			},
			after: func(t *testing.T, resp domain.LLMResponse) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
				defer cancel()
				var logModel dao.LLMRecord
				err := s.db.WithContext(ctx).Where("uid = ?", 124).First(&logModel).Error
				require.NoError(t, err)
				s.assertLog(dao.LLMRecord{
					Id:          1,
					Tid:         "11",
					Uid:         124,
					Biz:         domain.BizQuestionExamine,
					KnowledgeId: knowledgeId,
					Input: sqlx.JsonColumn[[]string]{
						Valid: true,
						Val: []string{
							"问题1",
							"问题1内容",
							"用户输入1",
						},
					},
					Status:         domain.RecordStatusFailed.ToUint8(),
					PromptTemplate: sqlx.NewNullString("这是问题 %s，这是问题内容 %s，这是用户输入 %s"),
				}, logModel)
			},
			assertFunc: assert.Error,
		},
		{
			name: "llm 调用失败",
			req: domain.LLMRequest{
				Biz: domain.BizQuestionExamine,
				Uid: 125,
				Tid: "11",
				Input: []string{
					"问题1",
					"问题1内容",
					"用户输入1",
				},
			},
			before: func(t *testing.T,
				ctrl *gomock.Controller) (*hdlmocks.MockHandler, credit.Service) {
				llmHdl := hdlmocks.NewMockHandler(ctrl)
				llmHdl.EXPECT().Handle(gomock.Any(), gomock.Any()).
					Return(domain.LLMResponse{}, errors.New("调用失败"))
				creditSvc := creditmocks.NewMockService(ctrl)
				creditSvc.EXPECT().GetCreditsByUID(gomock.Any(), gomock.Any()).Return(credit.Credit{
					TotalAmount: 1000,
				}, nil)
				return llmHdl, creditSvc
			},
			after: func(t *testing.T, resp domain.LLMResponse) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
				defer cancel()
				var logModel dao.LLMRecord
				err := s.db.WithContext(ctx).Where("uid = ?", 125).First(&logModel).Error
				require.NoError(t, err)
				s.assertLog(dao.LLMRecord{
					Id:          1,
					Tid:         "11",
					Uid:         125,
					Biz:         domain.BizQuestionExamine,
					Tokens:      100,
					Amount:      100,
					KnowledgeId: knowledgeId,
					Input: sqlx.JsonColumn[[]string]{
						Valid: true,
						Val: []string{
							"问题1",
							"问题1内容",
							"用户输入1",
						},
					},
					Status:         domain.CreditStatusFailed.ToUint8(),
					PromptTemplate: sqlx.NewNullString("这是问题 %s，这是问题内容 %s，这是用户输入 %s"),
					Answer:         sqlx.NewNullString("aians"),
				}, logModel)
				// 校验credit写入的内容是否正确
				var creditLogModel dao.LLMCredit
				err = s.db.WithContext(ctx).Where("id = ?", 1).First(&creditLogModel).Error
				require.NoError(t, err)
				s.assertCreditLog(dao.LLMCredit{
					Id:     1,
					Tid:    "11",
					Uid:    125,
					Biz:    domain.BizQuestionExamine,
					Amount: 100,
					Status: domain.RecordStatusFailed.ToUint8(),
				}, creditLogModel)
			},
			assertFunc: assert.Error,
		},
		{
			name: "积分足够，扣款失败",
			req: domain.LLMRequest{
				Biz: domain.BizQuestionExamine,
				Uid: 126,
				Tid: "11",
				Input: []string{
					"问题1",
					"问题1内容",
					"用户输入1",
				},
			},
			assertFunc: assert.Error,
			before: func(t *testing.T,
				ctrl *gomock.Controller) (*hdlmocks.MockHandler, credit.Service) {
				llmHdl := hdlmocks.NewMockHandler(ctrl)
				llmHdl.EXPECT().Handle(gomock.Any(), gomock.Any()).
					Return(domain.LLMResponse{
						Tokens: 100,
						Amount: 100,
						Answer: "aians",
					}, nil)
				creditSvc := creditmocks.NewMockService(ctrl)
				creditSvc.EXPECT().GetCreditsByUID(gomock.Any(), gomock.Any()).Return(credit.Credit{
					TotalAmount: 1000,
				}, nil)
				creditSvc.EXPECT().TryDeductCredits(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, c credit.Credit) (int64, error) {
					c.Logs[0].Key = ""
					c.Logs[0].BizId = 0
					assert.Equal(t, credit.Credit{
						Uid: 126,
						Logs: []credit.CreditLog{
							{
								ChangeAmount: 100,
								Uid:          126,
								Biz:          "ai-llm",
								Desc:         "ai-llm服务",
							},
						},
					}, c)
					return 0, errors.New("mock db error")
				})
				return llmHdl, creditSvc
			},
			after: func(t *testing.T, resp domain.LLMResponse) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
				defer cancel()
				// 校验response写入的内容是否正确
				assert.Equal(t, domain.LLMResponse{
					Tokens: 100,
					Amount: 100,
					Answer: "aians",
				}, resp)
				var logModel dao.LLMRecord
				err := s.db.WithContext(ctx).Where("uid = ?", 126).First(&logModel).Error
				require.NoError(t, err)
				s.assertLog(dao.LLMRecord{
					Id:          1,
					Tid:         "11",
					Uid:         126,
					Biz:         domain.BizQuestionExamine,
					Tokens:      100,
					Amount:      100,
					KnowledgeId: knowledgeId,
					Input: sqlx.JsonColumn[[]string]{
						Valid: true,
						Val: []string{
							"问题1",
							"问题1内容",
							"用户输入1",
						},
					},
					Status:         domain.RecordStatusFailed.ToUint8(),
					PromptTemplate: sqlx.NewNullString("这是问题 %s，这是问题内容 %s，这是用户输入 %s"),
					Answer:         sqlx.NewNullString("aians"),
				}, logModel)
				// 校验credit写入的内容是否正确
				var creditLogModel dao.LLMCredit
				err = s.db.WithContext(ctx).Where("id = ?", 1).First(&creditLogModel).Error
				require.NoError(t, err)
				s.assertCreditLog(dao.LLMCredit{
					Id:     1,
					Tid:    "11",
					Uid:    126,
					Biz:    domain.BizQuestionExamine,
					Amount: 100,
					Status: domain.CreditStatusFailed.ToUint8(),
				}, creditLogModel)
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
			defer cancel()
			mockHdl, mockCredit := tc.before(t, ctrl)
			mou, err := startup.InitModule(s.db, mockHdl, nil, nil, &credit.Module{Svc: mockCredit}, nil)
			require.NoError(t, err)
			resp, err := mou.Svc.Invoke(ctx, tc.req)
			tc.assertFunc(t, err)
			if err != nil {
				return
			}
			tc.after(t, resp)
		})
	}
}

func (s *LLMServiceSuite) TestHandler_Ask() {
	testCases := []struct {
		name       string
		req        web.LLMRequest
		before     func(t *testing.T, ctrl *gomock.Controller) (*hdlmocks.MockHandler, credit.Service)
		assertFunc assert.ErrorAssertionFunc
		after      func(t *testing.T, resp web.LLMResponse)
		wantCode   int
	}{
		{
			name: "八股文web-成功",
			req: web.LLMRequest{
				Biz: domain.BizQuestionExamine,
				Input: []string{
					"问题1",
					"问题1内容",
					"用户输入1",
				},
			},
			assertFunc: assert.NoError,
			before: func(t *testing.T,
				ctrl *gomock.Controller) (*hdlmocks.MockHandler, credit.Service) {
				llmHdl := hdlmocks.NewMockHandler(ctrl)
				llmHdl.EXPECT().Handle(gomock.Any(), gomock.Any()).
					Return(domain.LLMResponse{
						Tokens: 100,
						Amount: 100,
						Answer: "aians",
					}, nil)
				creditSvc := creditmocks.NewMockService(ctrl)
				creditSvc.EXPECT().GetCreditsByUID(gomock.Any(), gomock.Any()).Return(credit.Credit{
					TotalAmount: 1000,
				}, nil)
				creditSvc.EXPECT().TryDeductCredits(gomock.Any(), gomock.Any()).Return(11, nil)
				creditSvc.EXPECT().ConfirmDeductCredits(gomock.Any(), int64(123), int64(11)).Return(nil)
				return llmHdl, creditSvc
			},
			after: func(t *testing.T, resp web.LLMResponse) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
				defer cancel()
				// 校验response写入的内容是否正确
				assert.Equal(t, web.LLMResponse{
					Amount:    100,
					RawResult: "aians",
				}, resp)

				var logModel dao.LLMRecord
				err := s.db.WithContext(ctx).Where("id = ?", 1).First(&logModel).Error
				require.NoError(t, err)
				assert.True(t, logModel.Tid != "")
				s.assertLog(dao.LLMRecord{
					Id:          1,
					Uid:         123,
					Tid:         logModel.Tid,
					Biz:         domain.BizQuestionExamine,
					Tokens:      100,
					Amount:      100,
					KnowledgeId: knowledgeId,
					Input: sqlx.JsonColumn[[]string]{
						Valid: true,
						Val: []string{
							"问题1",
							"问题1内容",
							"用户输入1",
						},
					},
					Status:         1,
					PromptTemplate: sqlx.NewNullString("这是问题 %s，这是问题内容 %s，这是用户输入 %s"),
					Answer:         sqlx.NewNullString("aians"),
				}, logModel)
				// 校验credit写入的内容是否正确
				var creditLogModel dao.LLMCredit
				err = s.db.WithContext(ctx).Where("id = ?", 1).First(&creditLogModel).Error
				require.NoError(t, err)
				assert.True(t, logModel.Tid != "")

				s.assertCreditLog(dao.LLMCredit{
					Id:     1,
					Tid:    logModel.Tid,
					Uid:    123,
					Biz:    domain.BizQuestionExamine,
					Amount: 100,
					Status: 1,
				}, creditLogModel)
			},
			wantCode: 200,
		},
	}
	for _, tc := range testCases {
		tc := tc
		s.T().Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockHdl, mockCredit := tc.before(t, ctrl)
			mou, err := startup.InitModule(s.db, mockHdl, nil, nil, &credit.Module{Svc: mockCredit}, nil)
			require.NoError(t, err)
			req, err := http.NewRequest(http.MethodPost,
				"/ai/ask", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			econf.Set("server", map[string]any{"contextTimeout": "1s"})

			server := egin.Load("server").Build()
			server.Use(func(ctx *gin.Context) {
				ctx.Set(session.CtxSessionKey,
					session.NewMemorySession(session.Claims{
						Uid: 123,
					}))
			})
			mou.Hdl.MemberRoutes(server.Engine)
			recorder := test.NewJSONResponseRecorder[web.LLMResponse]()
			server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			tc.after(t, recorder.MustScan().Data)
		})
	}
}

func (s *LLMServiceSuite) TestHandler_AnalysisJD() {
	testCases := []struct {
		name       string
		req        web.JDRequest
		before     func(t *testing.T, ctrl *gomock.Controller) (*hdlmocks.MockHandler, credit.Service)
		assertFunc assert.ErrorAssertionFunc
		after      func(t *testing.T, resp web.JDResponse)
		wantCode   int
	}{
		{
			name: "岗位测评",
			req: web.JDRequest{
				JD: "我们招聘一个go工程师",
			},
			assertFunc: assert.NoError,
			before: func(t *testing.T,
				ctrl *gomock.Controller) (*hdlmocks.MockHandler, credit.Service) {
				llmHdl := hdlmocks.NewMockHandler(ctrl)
				llmHdl.EXPECT().Handle(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, request domain.LLMRequest) (domain.LLMResponse, error) {
						switch request.Biz {
						case domain.AnalysisJDTech:
							return domain.LLMResponse{
								Tokens: 1000,
								Amount: 100,
								Answer: `{"score":6, "summary":["这是技术前景"]}`,
							}, nil
						case domain.AnalysisJDBiz:
							return domain.LLMResponse{
								Tokens: 100,
								Amount: 200,
								Answer: `{"score":7, "summary":["这是业务前景"]}`,
							}, nil
						case domain.AnalysisJDPosition:
							return domain.LLMResponse{
								Tokens: 100,
								Amount: 100,
								Answer: `{"score":8, "summary":["这是公司地位"]}`,
							}, nil
						case domain.AnalysisJDSubtext:
							return domain.LLMResponse{
								Tokens: 100,
								Amount: 100,
								Answer: `这是我的分析`,
							}, nil
						default:
							return domain.LLMResponse{}, errors.New("unknown biz")
						}
					}).AnyTimes()
				creditSvc := creditmocks.NewMockService(ctrl)
				creditSvc.EXPECT().GetCreditsByUID(gomock.Any(), gomock.Any()).Return(credit.Credit{
					TotalAmount: 200000,
				}, nil).AnyTimes()
				creditSvc.EXPECT().TryDeductCredits(gomock.Any(), gomock.Any()).Return(11, nil).AnyTimes()
				creditSvc.EXPECT().ConfirmDeductCredits(gomock.Any(), int64(123), int64(11)).Return(nil).AnyTimes()
				return llmHdl, creditSvc
			},
			after: func(t *testing.T, resp web.JDResponse) {
				// 校验response写入的内容是否正确
				assert.Equal(t, web.JDResponse{
					Amount: 500,
					TechScore: web.JDEvaluation{
						Score:    6,
						Analysis: "- 这是技术前景",
					},
					BizScore: web.JDEvaluation{
						Score:    7,
						Analysis: "- 这是业务前景",
					},
					PosScore: web.JDEvaluation{
						Score:    8,
						Analysis: "- 这是公司地位",
					},
					Subtext: "这是我的分析",
				}, resp)

			},
			wantCode: 200,
		},
	}
	for _, tc := range testCases {
		tc := tc
		s.T().Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockHdl, mockCredit := tc.before(t, ctrl)
			mou, err := startup.InitModule(s.db, mockHdl, nil, nil, &credit.Module{Svc: mockCredit}, nil)
			require.NoError(t, err)
			req, err := http.NewRequest(http.MethodPost,
				"/ai/analysis_jd", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			econf.Set("server", map[string]any{"contextTimeout": "1s"})
			server := egin.Load("server").Build()
			server.Use(func(ctx *gin.Context) {
				ctx.Set(session.CtxSessionKey,
					session.NewMemorySession(session.Claims{
						Uid: 123,
					}))
			})
			mou.Hdl.MemberRoutes(server.Engine)
			recorder := test.NewJSONResponseRecorder[web.JDResponse]()
			server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			tc.after(t, recorder.MustScan().Data)
			err = s.db.Exec("TRUNCATE TABLE `llm_records`").Error
			require.NoError(s.T(), err)
			err = s.db.Exec("TRUNCATE TABLE `llm_credits`").Error
			require.NoError(s.T(), err)
		})
	}
}

func (s *LLMServiceSuite) TestHandler_Stream() {
	// 流式输出
	testcases := []struct {
		name     string
		req      web.LLMRequest
		before   func(t *testing.T, ctrl *gomock.Controller) *streamhdlmocks.MockStreamHandler
		wantEvts []web.Event
		wantCode int
	}{
		{
			name: "流式接口",
			req: web.LLMRequest{
				Biz: domain.BizQuestionExamine,
				Input: []string{
					"请说一下什么是deepseek ?",
				},
			},
			before: func(t *testing.T, ctrl *gomock.Controller) *streamhdlmocks.MockStreamHandler {
				mockStreamHandler := streamhdlmocks.NewMockStreamHandler(ctrl)
				mockStreamHandler.EXPECT().StreamHandle(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, req domain.LLMRequest) (chan domain.StreamEvent, error) {
					require.True(t, req.Tid != "")
					req.Tid = ""
					assert.Equal(t, domain.LLMRequest{
						Biz:   domain.BizQuestionExamine,
						Uid:   123,
						Input: []string{"请说一下什么是deepseek ?"},
					}, req)

					events := make(chan domain.StreamEvent, 5)
					go func() {
						defer close(events)
						events <- domain.StreamEvent{ReasoningContent: "reasoning1"}
						time.Sleep(100 * time.Millisecond)
						events <- domain.StreamEvent{ReasoningContent: "reasoning2"}
						time.Sleep(100 * time.Millisecond)
						events <- domain.StreamEvent{Content: "msg3"}
						events <- domain.StreamEvent{Content: "msg4"}
						events <- domain.StreamEvent{Done: true}
					}()
					return events, nil
				})
				return mockStreamHandler
			},
			wantEvts: []web.Event{
				{Type: "msg", Data: web.EvtMsg{
					ReasoningContent: "reasoning1",
				}},
				{Type: "msg", Data: web.EvtMsg{
					ReasoningContent: "reasoning2",
				}},
				{Type: "msg", Data: web.EvtMsg{
					Content: "msg3",
				}},
				{Type: "msg", Data: web.EvtMsg{
					Content: "msg4",
				}},
				{Type: "end"},
			},
			wantCode: 200,
		},
	}
	for _, tc := range testcases {
		tc := tc
		s.T().Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			streamMockHdl := tc.before(t, ctrl)
			mou, err := startup.InitModule(s.db, nil, streamMockHdl, nil, &credit.Module{Svc: nil}, nil)
			require.NoError(t, err)
			req, err := http.NewRequest(http.MethodPost,
				"/ai/stream", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			econf.Set("server", map[string]any{"contextTimeout": "1s"})

			server := egin.Load("server").Build()
			server.Use(func(ctx *gin.Context) {
				ctx.Set(session.CtxSessionKey,
					session.NewMemorySession(session.Claims{
						Uid: 123,
					}))
			})
			mou.Hdl.MemberRoutes(server.Engine)
			w := httptest.NewRecorder()
			server.ServeHTTP(w, req)
			// 6. 验证响应
			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
			evts := parseSSEResponse(t, w.Body)
			assert.Equal(t, tc.wantEvts, evts)
		})
	}
}

func parseSSEResponse(t *testing.T, body *bytes.Buffer) []web.Event {
	var events []web.Event
	for _, line := range strings.Split(body.String(), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var evt web.Event
		err := json.Unmarshal([]byte(line), &evt)
		require.NoError(t, err)
		events = append(events, evt)
	}
	return events
}

func (s *LLMServiceSuite) assertLog(wantLog dao.LLMRecord, actual dao.LLMRecord) {
	require.True(s.T(), actual.Ctime != 0)
	require.True(s.T(), actual.Utime != 0)
	actual.Ctime = 0
	actual.Utime = 0
	assert.Equal(s.T(), wantLog, actual)
}

func (s *LLMServiceSuite) assertCreditLog(wantLog dao.LLMCredit, actual dao.LLMCredit) {
	require.True(s.T(), actual.Ctime != 0)
	require.True(s.T(), actual.Utime != 0)
	actual.Ctime = 0
	actual.Utime = 0
	assert.Equal(s.T(), wantLog, actual)
}
