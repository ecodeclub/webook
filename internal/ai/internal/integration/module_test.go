//go:build e2e

package integration

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/ecodeclub/webook/internal/ai/internal/domain"
	"github.com/ecodeclub/webook/internal/ai/internal/integration/startup"
	"github.com/ecodeclub/webook/internal/ai/internal/repository/dao"
	gptmocks "github.com/ecodeclub/webook/internal/ai/internal/service/handler/gpt/mocks"
	"github.com/ecodeclub/webook/internal/ai/internal/service/handler/gpt/sdk"
	"github.com/ecodeclub/webook/internal/credit"
	creditmocks "github.com/ecodeclub/webook/internal/credit/mocks"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/ego-component/egorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type GptSuite struct {
	suite.Suite
	logDao dao.GPTLogDAO
	db     *egorm.Component
}

func TestGptSuite(t *testing.T) {
	suite.Run(t, new(GptSuite))
}

func (g *GptSuite) SetupSuite() {
	db := testioc.InitDB()
	g.db = db
	g.logDao = dao.NewGPTLogDAO(db)
}
func (s *GptSuite) TearDownSuite() {
	err := s.db.Exec("DROP TABLE `gpt_logs`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("DROP TABLE `gpt_credit_logs`").Error
	require.NoError(s.T(), err)
}

func (s *GptSuite) TearDownTest() {
	err := s.db.Exec("TRUNCATE TABLE `gpt_logs`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE `gpt_credit_logs`").Error
	require.NoError(s.T(), err)
}

func (g *GptSuite) TestService() {
	t := g.T()
	tesecases := []struct {
		name       string
		req        domain.GPTRequest
		newSvcFunc func(t *testing.T, ctrl *gomock.Controller) credit.Service
		newAiFunc  func(t *testing.T, ctrl *gomock.Controller) sdk.GPTSdk
		assertFunc assert.ErrorAssertionFunc
		after      func(t *testing.T, resp domain.GPTResponse)
	}{
		{
			name: "成功访问",
			req: domain.GPTRequest{
				Biz: "simple",
				Uid: 123,
				Tid: "11",
				Input: []string{
					"nihao",
				},
			},
			newAiFunc: func(t *testing.T, ctrl *gomock.Controller) sdk.GPTSdk {
				mockAiSdk := gptmocks.NewMockGPTSdk(ctrl)
				mockAiSdk.EXPECT().Invoke(gomock.Any(), gomock.Any()).Return(100, "aians", nil)
				return mockAiSdk
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) credit.Service {
				mockCreditSvc := creditmocks.NewMockService(ctrl)
				mockCreditSvc.EXPECT().GetCreditsByUID(gomock.Any(), int64(123)).Return(credit.Credit{
					TotalAmount:       1000,
					LockedTotalAmount: 0,
				}, nil)
				mockCreditSvc.EXPECT().AddCredits(gomock.Any(), gomock.Any()).Do(func(ctx context.Context, cre credit.Credit) error {
					if cre.Uid != 123 {
						return errors.New("incorrect uid")
					}
					if len(cre.Logs) <= 0 {
						return errors.New("incorrect logs")
					}
					l := cre.Logs[0]
					if l.ChangeAmount != 100 || l.Biz != "ai-gpt" || l.BizId <= 0 || l.Desc == "" {
						return errors.New("incorrect logs")
					}
					return nil
				})
				return mockCreditSvc
			},
			assertFunc: assert.NoError,
			after: func(t *testing.T, resp domain.GPTResponse) {
				// 校验response写入的内容是否正确
				assert.Equal(t, domain.GPTResponse{
					Tokens: 100,
					Amount: 100,
					Answer: "aians",
				}, resp)
				logModel, err := g.logDao.FirstLog(context.Background(), 1)
				require.NoError(t, err)
				g.assertLog(&dao.GptLog{
					Id:     1,
					Tid:    "11",
					Uid:    123,
					Biz:    "simple",
					Tokens: 100,
					Amount: 100,
					Status: 1,
					Prompt: sql.NullString{
						Valid:  true,
						String: "[\"nihao\"]",
					},
					Answer: sql.NullString{
						Valid:  true,
						String: "aians",
					},
				}, logModel)
				// 校验credit写入的内容是否正确
				creditLogModel, err := g.logDao.FirstCreditLog(context.Background(), 1)
				require.NoError(t, err)
				g.assertCreditLog(&dao.GptCreditLog{
					Id:     1,
					Tid:    "11",
					Uid:    123,
					Biz:    "simple",
					Tokens: 100,
					Amount: 100,
					Credit: 100,
					Status: 1,
					Prompt: sql.NullString{
						Valid:  true,
						String: "[\"nihao\"]",
					},
					Answer: sql.NullString{
						Valid:  true,
						String: "aians",
					},
				}, creditLogModel)
			},
		},
		{
			name: "积分不足扣款失败",
			req: domain.GPTRequest{
				Biz: "simple",
				Uid: 123,
				Tid: "11",
				Input: []string{
					"nihao",
				},
			},
			newAiFunc: func(t *testing.T, ctrl *gomock.Controller) sdk.GPTSdk {
				mockAiSdk := gptmocks.NewMockGPTSdk(ctrl)

				return mockAiSdk
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) credit.Service {
				mockCreditSvc := creditmocks.NewMockService(ctrl)
				mockCreditSvc.EXPECT().GetCreditsByUID(gomock.Any(), int64(123)).Return(credit.Credit{
					TotalAmount:       1,
					LockedTotalAmount: 0,
				}, nil)

				return mockCreditSvc
			},
			assertFunc: assert.Error,
		},
		{
			name: "creditSvc调用失败",
			req: domain.GPTRequest{
				Biz: "simple",
				Uid: 123,
				Tid: "11",
				Input: []string{
					"nihao",
				},
			},
			newAiFunc: func(t *testing.T, ctrl *gomock.Controller) sdk.GPTSdk {
				mockAiSdk := gptmocks.NewMockGPTSdk(ctrl)
				mockAiSdk.EXPECT().Invoke(gomock.Any(), gomock.Any()).Return(100, "aians", nil)
				return mockAiSdk
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) credit.Service {
				mockCreditSvc := creditmocks.NewMockService(ctrl)
				mockCreditSvc.EXPECT().GetCreditsByUID(gomock.Any(), int64(123)).Return(credit.Credit{
					TotalAmount:       1000,
					LockedTotalAmount: 0,
				}, nil)
				mockCreditSvc.EXPECT().AddCredits(gomock.Any(), gomock.Any()).Return(errors.New("服务内部错误"))
				return mockCreditSvc
			},
			assertFunc: assert.Error,
		},
	}
	for _, tc := range tesecases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			aiSdk := tc.newAiFunc(t, ctrl)
			creditSvc := tc.newSvcFunc(t, ctrl)
			mou, err := startup.InitModule(aiSdk, creditSvc)
			require.NoError(t, err)
			resp, err := mou.Svc.Invoke(context.Background(), tc.req)
			tc.assertFunc(t, err)
			if err != nil {
				return
			}
			tc.after(t, resp)
		})
	}
}

func (g *GptSuite) assertLog(wantLog *dao.GptLog, actual *dao.GptLog) {
	require.True(g.T(), actual.Ctime != 0)
	require.True(g.T(), actual.Utime != 0)
	actual.Ctime = 0
	actual.Utime = 0
	assert.Equal(g.T(), wantLog, actual)
}

func (g *GptSuite) assertCreditLog(wantLog *dao.GptCreditLog, actual *dao.GptCreditLog) {
	require.True(g.T(), actual.Ctime != 0)
	require.True(g.T(), actual.Utime != 0)
	actual.Ctime = 0
	actual.Utime = 0
	assert.Equal(g.T(), wantLog, actual)
}
