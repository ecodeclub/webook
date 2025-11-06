package integration

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"github.com/ecodeclub/ginx"

	"github.com/ecodeclub/ekit/iox"
	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ginx/session"
	st "github.com/ecodeclub/webook/internal/bff/internal/integration/startup"
	"github.com/ecodeclub/webook/internal/bff/internal/web"
	"github.com/ecodeclub/webook/internal/cases"
	casemocks "github.com/ecodeclub/webook/internal/cases/mocks"
	"github.com/ecodeclub/webook/internal/interactive"
	intrmocks "github.com/ecodeclub/webook/internal/interactive/mocks"
	baguwen "github.com/ecodeclub/webook/internal/question"
	quemocks "github.com/ecodeclub/webook/internal/question/mocks"
	"github.com/ecodeclub/webook/internal/test"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/econf"
	"github.com/gotomicro/ego/server/egin"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type CollectionHandlerTestSuite struct {
	suite.Suite
	server *egin.Component
}

const uid = 123

func (c *CollectionHandlerTestSuite) SetupSuite() {
	ctrl := gomock.NewController(c.T())
	queSvc := quemocks.NewMockService(ctrl)
	queSetSvc := quemocks.NewMockQuestionSetService(ctrl)
	intrSvc := intrmocks.NewMockService(ctrl)
	intrSvc.EXPECT().CollectionInfo(gomock.Any(), int64(uid), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, uid int64, id int64, biz string, offset int, limit int) ([]interactive.CollectionRecord, int, error) {

		switch biz {
		case web.CaseBiz:
			return []interactive.CollectionRecord{
				{
					Biz:  web.CaseBiz,
					Case: 1,
				},
			}, 1, nil
		case web.CaseSetBiz:
			return []interactive.CollectionRecord{
				{
					Biz:     web.CaseSetBiz,
					CaseSet: 5,
				},
			}, 1, nil
		case web.QuestionBiz:
			return []interactive.CollectionRecord{
				{
					Biz:      web.QuestionBiz,
					Question: 2,
				},
			}, 1, nil
		case web.QuestionSetBiz:
			return []interactive.CollectionRecord{
				{
					Biz:         web.QuestionSetBiz,
					QuestionSet: 3,
				},
				{
					Biz:         web.QuestionSetBiz,
					QuestionSet: 4,
				},
			}, 2, nil
		case "":
			return []interactive.CollectionRecord{
				{
					Biz:  web.CaseBiz,
					Case: 1,
				},
				{
					Biz:      web.QuestionBiz,
					Question: 2,
				},
				{
					Biz:         web.QuestionSetBiz,
					QuestionSet: 3,
				},
				{
					Biz:         web.QuestionSetBiz,
					QuestionSet: 4,
				},
				{
					Biz:     web.CaseSetBiz,
					CaseSet: 5,
				},
			}, 5, nil
		default:
			return nil, 0, errors.New("unknown biz")
		}

	}).AnyTimes()
	queSvc.EXPECT().GetPubByIDs(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, ids []int64) ([]baguwen.Question, error) {
			return slice.Map(ids, func(idx int, src int64) baguwen.Question {
				return baguwen.Question{
					Id:    src,
					Title: "这是题目" + strconv.FormatInt(src, 10),
				}
			}), nil
		}).AnyTimes()
	queSetSvc.EXPECT().GetByIDsWithQuestion(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, ids []int64) ([]baguwen.QuestionSet, error) {
		return slice.Map(ids, func(idx int, src int64) baguwen.QuestionSet {
			return baguwen.QuestionSet{
				Id:    src,
				Title: fmt.Sprintf("这是题集%d", src),
				Questions: []baguwen.Question{
					{
						Id:    src*10 + src,
						Title: fmt.Sprintf("这是题目%d", src*10+src),
					},
					{
						Id:    src*11 + src,
						Title: fmt.Sprintf("这是题目%d", src*11+src),
					},
				},
			}
		}), nil
	}).AnyTimes()

	caseExamSvc := casemocks.NewMockExamineService(ctrl)
	caseExamSvc.EXPECT().GetResults(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, uid int64, ids []int64) (map[int64]cases.ExamineResult, error) {
			res := make(map[int64]cases.ExamineResult, len(ids))
			for _, id := range ids {
				res[id] = cases.ExamineResult{
					Cid: id,
					// 偶数不通过，基数通过
					Result: cases.ExamineResultEnum(id % 2),
				}
			}
			return res, nil
		}).AnyTimes()

	caseSvc := casemocks.NewMockService(ctrl)
	caseSvc.EXPECT().GetPubByIDs(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, ids []int64) ([]cases.Case, error) {
			return slice.Map(ids, func(idx int, src int64) cases.Case {
				return cases.Case{
					Id:    src,
					Title: "这是案例" + strconv.FormatInt(src, 10),
				}
			}), nil
		}).AnyTimes()
	caseSetSvc := casemocks.NewMockCaseSetService(ctrl)
	caseSetSvc.EXPECT().GetByIdsWithCases(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, ids []int64) ([]cases.CaseSet, error) {
			return slice.Map(ids, func(idx int, src int64) cases.CaseSet {
				return cases.CaseSet{
					ID:    src,
					Title: fmt.Sprintf("这是案例集%d", src),
					Cases: []cases.Case{
						{
							Id:    src*10 + 1,
							Title: fmt.Sprintf("这是案例%d", src*10+1),
						},
						{
							Id:    src*10 + 2,
							Title: fmt.Sprintf("这是案例%d", src*10+2),
						},
					},
				}
			}), nil
		}).AnyTimes()
	handler, _ := st.InitHandler(&interactive.Module{Svc: intrSvc},
		&cases.Module{Svc: caseSvc, SetSvc: caseSetSvc, ExamineSvc: caseExamSvc},
		&baguwen.Module{Svc: queSvc, SetSvc: queSetSvc})
	econf.Set("server", map[string]any{"contextTimeout": "1s"})
	server := egin.Load("server").Build()
	server.Use(func(ctx *gin.Context) {
		ctx.Set("_session", session.NewMemorySession(session.Claims{
			Uid:  uid,
			Data: map[string]string{"creator": "true"},
		}))
	})
	handler.PrivateRoutes(server.Engine)
	c.server = server
}

func (c *CollectionHandlerTestSuite) Test_Handler() {
	t := c.T()

	testCases := []struct {
		name string
		req  web.CollectionInfoReq

		wantCode int
		wantResp ginx.DataList[web.CollectionRecord]
	}{
		{
			name: "获取全部收藏夹记录成功",
			req: web.CollectionInfoReq{
				ID:     1,
				Biz:    "",
				Offset: 0,
				Limit:  10,
			},
			wantCode: http.StatusOK,
			wantResp: ginx.DataList[web.CollectionRecord]{
				List: []web.CollectionRecord{
					{
						Case: web.Case{
							ID:    1,
							Title: "这是案例1",
						},
					},
					{
						Question: web.Question{
							ID:    2,
							Title: "这是题目2",
						},
					},
					{
						QuestionSet: web.QuestionSet{
							ID:    3,
							Title: "这是题集3",
							Questions: []web.Question{
								{
									ID:    33,
									Title: "这是题目33",
								},
								{
									ID:    36,
									Title: "这是题目36",
								},
							},
						},
					},
					{
						QuestionSet: web.QuestionSet{
							ID:    4,
							Title: "这是题集4",
							Questions: []web.Question{
								{
									ID:    44,
									Title: "这是题目44",
								},
								{
									ID:    48,
									Title: "这是题目48",
								},
							},
						},
					},
					{
						CaseSet: web.CaseSet{
							ID:    5,
							Title: "这是案例集5",
							Cases: []web.Case{
								{
									ID:            51,
									ExamineResult: 1,
								},
								{
									ID: 52,
								},
							},
						},
					},
				},
				Total: 5,
			},
		},
		{
			name: "仅获取收藏夹中的Case记录成功",
			req: web.CollectionInfoReq{
				ID:     1,
				Biz:    web.CaseBiz,
				Offset: 0,
				Limit:  10,
			},
			wantCode: http.StatusOK,
			wantResp: ginx.DataList[web.CollectionRecord]{
				List: []web.CollectionRecord{
					{
						Case: web.Case{
							ID:    1,
							Title: "这是案例1",
						},
					},
				},
				Total: 1,
			},
		},
		{
			name: "仅获取收藏夹中的CaseSet记录成功",
			req: web.CollectionInfoReq{
				ID:     1,
				Biz:    web.CaseSetBiz,
				Offset: 0,
				Limit:  10,
			},
			wantCode: http.StatusOK,
			wantResp: ginx.DataList[web.CollectionRecord]{
				List: []web.CollectionRecord{
					{
						CaseSet: web.CaseSet{
							ID:    5,
							Title: "这是案例集5",
							Cases: []web.Case{
								{
									ID:            51,
									ExamineResult: 1,
								},
								{
									ID: 52,
								},
							},
						},
					},
				},
				Total: 1,
			},
		},
		{
			name: "仅获取收藏夹中的Question记录成功",
			req: web.CollectionInfoReq{
				ID:     1,
				Biz:    web.QuestionBiz,
				Offset: 0,
				Limit:  10,
			},
			wantCode: http.StatusOK,
			wantResp: ginx.DataList[web.CollectionRecord]{
				List: []web.CollectionRecord{
					{
						Question: web.Question{
							ID:    2,
							Title: "这是题目2",
						},
					},
				},
				Total: 1,
			},
		},
		{
			name: "仅获取收藏夹中的QuestionSet记录成功",
			req: web.CollectionInfoReq{
				ID:     1,
				Biz:    web.QuestionSetBiz,
				Offset: 0,
				Limit:  10,
			},
			wantCode: http.StatusOK,
			wantResp: ginx.DataList[web.CollectionRecord]{
				List: []web.CollectionRecord{
					{
						QuestionSet: web.QuestionSet{
							ID:    3,
							Title: "这是题集3",
							Questions: []web.Question{
								{
									ID:    33,
									Title: "这是题目33",
								},
								{
									ID:    36,
									Title: "这是题目36",
								},
							},
						},
					},
					{
						QuestionSet: web.QuestionSet{
							ID:    4,
							Title: "这是题集4",
							Questions: []web.Question{
								{
									ID:    44,
									Title: "这是题目44",
								},
								{
									ID:    48,
									Title: "这是题目48",
								},
							},
						},
					},
				},
				Total: 2,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost,
				"/interactive/collection/records", iox.NewJSONReader(tc.req))
			require.NoError(t, err)
			req.Header.Set("content-type", "application/json")
			recorder := test.NewJSONResponseRecorder[ginx.DataList[web.CollectionRecord]]()
			c.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			require.Equal(t, tc.wantResp, recorder.MustScan().Data)
		})
	}
}

func TestCollectionHandler(t *testing.T) {
	suite.Run(t, new(CollectionHandlerTestSuite))
}
