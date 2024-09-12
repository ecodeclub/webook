package integration

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/ecodeclub/ekit/iox"
	"github.com/ecodeclub/ekit/sqlx"
	"github.com/ecodeclub/webook/internal/resume/internal/domain"
	"github.com/ecodeclub/webook/internal/resume/internal/web"
	"github.com/ecodeclub/webook/internal/test"
	"github.com/stretchr/testify/assert"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/cases"
	casemocks "github.com/ecodeclub/webook/internal/cases/mocks"
	"github.com/ecodeclub/webook/internal/resume/internal/integration/startup"
	"github.com/ecodeclub/webook/internal/resume/internal/repository/dao"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/ego-component/egorm"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/econf"
	"github.com/gotomicro/ego/server/egin"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const uid = 1235678

type ExperienceTestSuite struct {
	suite.Suite
	server *egin.Component
	db     *egorm.Component
	dao    dao.ExperienceDAO
	ctrl   *gomock.Controller
}

func (s *ExperienceTestSuite) TearDownTest() {
	err := s.db.Exec("TRUNCATE TABLE `experiences`").Error
	require.NoError(s.T(), err)
}

func (s *ExperienceTestSuite) SetupSuite() {
	ctrl := gomock.NewController(s.T())
	examSvc := casemocks.NewMockExamineService(ctrl)
	examSvc.EXPECT().GetResults(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, uid int64, ids []int64) (map[int64]cases.ExamineResult, error) {
		res := slice.Map(ids, func(idx int, src int64) cases.ExamineResult {
			return cases.ExamineResult{
				Cid:    src,
				Result: cases.ExamineResultEnum(src % 4),
			}
		})
		resMap := make(map[int64]cases.ExamineResult, len(res))
		for _, examRes := range res {
			resMap[examRes.Cid] = examRes
		}
		return resMap, nil
	}).AnyTimes()

	module := startup.InitModule(&cases.Module{
		ExamineSvc: examSvc,
	})
	econf.Set("server", map[string]any{"contextTimeout": "1s"})
	server := egin.Load("server").Build()

	server.Use(func(ctx *gin.Context) {
		ctx.Set("_session", session.NewMemorySession(session.Claims{
			Uid: uid,
		}))
	})

	module.ExperienceHdl.PrivateRoutes(server.Engine)
	s.server = server

	s.db = testioc.InitDB()
	err := dao.InitTables(s.db)
	require.NoError(s.T(), err)
	s.dao = dao.NewExperienceDAO(s.db)
}

func (s *ExperienceTestSuite) TestSave() {
	testCase := []struct {
		name     string
		before   func(t *testing.T)
		after    func(t *testing.T)
		req      web.Experience
		wantCode int
		wantResp test.Result[int64]
	}{
		{
			name: "创建工作经历",
			before: func(t *testing.T) {

			},
			after: func(t *testing.T) {
				_, cancel := context.WithTimeout(context.Background(), time.Second*3)
				defer cancel()

				var experiences []dao.Experience
				err := s.db.Where("uid = ?", uid).Order("start_time desc").Find(&experiences).Error

				require.NoError(t, err)
				assert.Len(t, experiences, 1)

				for i, _ := range experiences {
					assert.True(t, experiences[i].Utime > 0)
					experiences[i].Utime = 0
					assert.True(t, experiences[i].Ctime > 0)
					experiences[i].Ctime = 0
					assert.True(t, experiences[i].ID > 0)
					experiences[i].ID = 0
				}

				assert.Equal(t, dao.Experience{
					Uid:       uid,
					StartTime: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli(),
					EndTime:   time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli(),
					Title:     "测试工程师",
					Responsibilities: sqlx.JsonColumn[[]domain.Responsibility]{
						Valid: true,
						Val: []domain.Responsibility{
							domain.Responsibility{
								Type:    "aaaaaaa",
								Content: "bbbbbbbbfjsdfjlajlfjadfjladjfldjfldjflajdf",
							}, {
								Type:    "aaaaaaa",
								Content: "bbbbbbbbfjsdfjlajlfjadfjladjfldjfldjflajdf",
							},
						},
					},
					Accomplishments: sqlx.JsonColumn[[]domain.Accomplishment]{
						Valid: true,
						Val:   []domain.Accomplishment{},
					},
					Skills: sqlx.JsonColumn[[]string]{
						Valid: true,
						Val: []string{
							"Jerry",
							"Kelly",
						},
					},
				}, experiences[0])
			},
			req: web.Experience{

				Start: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				End:   time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
				Title: "测试工程师",
				Responsibilities: []web.Responsibility{
					web.Responsibility{
						Type:    "aaaaaaa",
						Content: "bbbbbbbbfjsdfjlajlfjadfjladjfldjfldjflajdf",
					}, {
						Type:    "aaaaaaa",
						Content: "bbbbbbbbfjsdfjlajlfjadfjladjfldjfldjflajdf",
					},
				},
				Accomplishments: []web.Accomplishment{},
				Skills: []string{
					"Jerry",
					"Kelly",
				},
			},
			wantCode: http.StatusOK,
			wantResp: test.Result[int64]{
				Data: 1,
			},
		}, {
			name: "修改工作经历",
			before: func(t *testing.T) {

			},
			after: func(t *testing.T) {
				_, cancel := context.WithTimeout(context.Background(), time.Second*3)
				defer cancel()

				var experience dao.Experience
				err := s.db.Where("id = ?", 1).Order("start_time desc").Find(&experience).Error

				require.NoError(t, err)

				assert.True(t, experience.Utime > 0)
				experience.Utime = 0
				assert.True(t, experience.Ctime > 0)
				experience.Ctime = 0
				assert.True(t, experience.ID > 0)
				experience.ID = 0

				assert.Equal(t, dao.Experience{
					Uid:       uid,
					StartTime: time.Date(2021, 2, 1, 0, 0, 0, 0, time.UTC).UnixMilli(),
					EndTime:   time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli(),
					Title:     "测试工程师",
					Responsibilities: sqlx.JsonColumn[[]domain.Responsibility]{
						Valid: true,
						Val: []domain.Responsibility{
							domain.Responsibility{
								Type:    "bbbbbbb",
								Content: "bbbbbbbbfjsdfjlajlfjadfjladjfldjfldjflajdf",
							},
						},
					},
					Accomplishments: sqlx.JsonColumn[[]domain.Accomplishment]{
						Valid: true,
						Val: []domain.Accomplishment{
							domain.Accomplishment{
								Type:    "aaaaaaa",
								Content: "jksjfsjfdlksdjflsjdlfdjsfjslfkls",
							},
						},
					},
					Skills: sqlx.JsonColumn[[]string]{
						Valid: true,
						Val: []string{
							"Jerry",
							"Kelly",
						},
					},
				}, experience)
			},
			req: web.Experience{
				Id:    1,
				Start: time.Date(2021, 2, 1, 0, 0, 0, 0, time.UTC),
				End:   time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
				Title: "测试工程师",
				Responsibilities: []web.Responsibility{
					web.Responsibility{
						Type:    "bbbbbbb",
						Content: "bbbbbbbbfjsdfjlajlfjadfjladjfldjfldjflajdf",
					},
				},
				Accomplishments: []web.Accomplishment{
					web.Accomplishment{
						Type:    "aaaaaaa",
						Content: "jksjfsjfdlksdjflsjdlfdjsfjslfkls",
					},
				},
				Skills: []string{
					"Jerry",
					"Kelly",
				},
			},
			wantCode: http.StatusOK,
			wantResp: test.Result[int64]{
				Data: 1,
			},
		},
	}

	for _, tc := range testCase {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/resume/experience/save", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[int64]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
			tc.after(t)

		})
	}
}

func (s *ExperienceTestSuite) TestDelete() {
	testCase := []struct {
		name     string
		before   func(t *testing.T)
		after    func(t *testing.T)
		req      web.Experience
		wantCode int
		wantResp test.Result[any]
	}{
		{
			name: "插入两条数据，删除第一条",
			before: func(t *testing.T) {
				experiences := []dao.Experience{
					dao.Experience{
						Uid:       uid,
						StartTime: time.Date(2021, 2, 1, 0, 0, 0, 0, time.UTC).UnixMilli(),
						EndTime:   time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli(),
						Title:     "测试工程师",
						Responsibilities: sqlx.JsonColumn[[]domain.Responsibility]{
							Valid: true,
							Val: []domain.Responsibility{
								domain.Responsibility{
									Type:    "bbbbbbb",
									Content: "bbbbbbbbfjsdfjlajlfjadfjladjfldjfldjflajdf",
								},
							},
						},
						Accomplishments: sqlx.JsonColumn[[]domain.Accomplishment]{
							Valid: true,
							Val: []domain.Accomplishment{
								domain.Accomplishment{
									Type:    "aaaaaaa",
									Content: "jksjfsjfdlksdjflsjdlfdjsfjslfkls",
								},
							},
						},
						Skills: sqlx.JsonColumn[[]string]{
							Valid: true,
							Val: []string{
								"Jerry",
								"Kelly",
							},
						},
					},
					{
						Uid:       uid,
						StartTime: time.Date(2021, 2, 1, 0, 0, 0, 0, time.UTC).UnixMilli(),
						EndTime:   time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli(),
						Title:     "测试工程师",
						Responsibilities: sqlx.JsonColumn[[]domain.Responsibility]{
							Valid: true,
							Val: []domain.Responsibility{
								domain.Responsibility{
									Type:    "bbbbbbb",
									Content: "bbbbbbbbfjsdfjlajlfjadfjladjfldjfldjflajdf",
								},
							},
						},
						Accomplishments: sqlx.JsonColumn[[]domain.Accomplishment]{
							Valid: true,
							Val: []domain.Accomplishment{
								domain.Accomplishment{
									Type:    "aaaaaaa",
									Content: "jksjfsjfdlksdjflsjdlfdjsfjslfkls",
								},
							},
						},
						Skills: sqlx.JsonColumn[[]string]{
							Valid: true,
							Val: []string{
								"Jerry",
								"Kelly",
							},
						},
					},
				}
				err := s.db.Create(experiences).Error
				require.NoError(t, err)

			},
			after: func(t *testing.T) {
				var experiences []dao.Experience
				err := s.db.Where("uid = ?", uid).Order("start_time desc").Find(&experiences).Error
				require.NoError(t, err)

				assert.Equal(t, len(experiences), 1)
				assert.Equal(t, experiences[0].ID, int64(2))
			},
			req: web.Experience{
				Id: 1,
			},
			wantCode: http.StatusOK,
			wantResp: test.Result[any]{
				Msg:  "success",
				Data: nil,
			},
		}, {
			name: "删除第二条数据",
			before: func(t *testing.T) {

			},
			after: func(t *testing.T) {
				var experiences []dao.Experience
				err := s.db.Where("uid = ?", uid).Order("start_time desc").Find(&experiences).Error
				require.NoError(t, err)

				assert.Equal(t, len(experiences), 0)
			},
			req: web.Experience{
				Id: 2,
			},
			wantCode: http.StatusOK,
			wantResp: test.Result[any]{
				Msg:  "success",
				Data: nil,
			},
		}, {
			name: "删除不存在的数据",
			before: func(t *testing.T) {

			},
			after: func(t *testing.T) {
				var experiences []dao.Experience
				err := s.db.Where("uid = ?", uid).Order("start_time desc").Find(&experiences).Error
				require.NoError(t, err)

				assert.Equal(t, len(experiences), 0)
			},
			req: web.Experience{
				Id: 2,
			},
			wantCode: http.StatusInternalServerError,
			wantResp: test.Result[any]{
				Code: 515001,
				Msg:  "系统错误",
				Data: nil,
			},
		},
	}

	for _, tc := range testCase {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/resume/experience/delete", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[any]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
			tc.after(t)

		})
	}
}

func (s *ExperienceTestSuite) TestList() {
	testCase := []struct {
		name     string
		before   func(t *testing.T)
		after    func(t *testing.T)
		wantCode int
		wantResp test.Result[[]web.Experience]
	}{
		{
			name: "查询所有的经历",
			before: func(t *testing.T) {
				experiences := []dao.Experience{
					dao.Experience{
						Uid:       uid,
						StartTime: time.Date(2019, 2, 1, 0, 0, 0, 0, time.UTC).UnixMilli(),
						EndTime:   time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli(),
						Title:     "测试工程师",
						Responsibilities: sqlx.JsonColumn[[]domain.Responsibility]{
							Valid: true,
							Val: []domain.Responsibility{
								domain.Responsibility{
									Type:    "bbbbbbb",
									Content: "bbbbbbbbfjsdfjlajlfjadfjladjfldjfldjflajdf",
								},
							},
						},
						Accomplishments: sqlx.JsonColumn[[]domain.Accomplishment]{
							Valid: true,
							Val: []domain.Accomplishment{
								domain.Accomplishment{
									Type:    "aaaaaaa",
									Content: "jksjfsjfdlksdjflsjdlfdjsfjslfkls",
								},
							},
						},
						Skills: sqlx.JsonColumn[[]string]{
							Valid: true,
							Val: []string{
								"Jerry",
								"Kelly",
							},
						},
					},
					{
						Uid:       uid,
						StartTime: time.Date(2020, 2, 1, 0, 0, 0, 0, time.UTC).UnixMilli(),
						EndTime:   time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli(),
						Title:     "测试工程师",
						Responsibilities: sqlx.JsonColumn[[]domain.Responsibility]{
							Valid: true,
							Val: []domain.Responsibility{
								domain.Responsibility{
									Type:    "bbbbbbb",
									Content: "bbbbbbbbfjsdfjlajlfjadfjladjfldjfldjflajdf",
								},
							},
						},
						Accomplishments: sqlx.JsonColumn[[]domain.Accomplishment]{
							Valid: true,
							Val: []domain.Accomplishment{
								domain.Accomplishment{
									Type:    "aaaaaaa",
									Content: "jksjfsjfdlksdjflsjdlfdjsfjslfkls",
								},
							},
						},
						Skills: sqlx.JsonColumn[[]string]{
							Valid: true,
							Val: []string{
								"Jerry",
								"Kelly",
							},
						},
					},
				}
				err := s.db.Create(experiences).Error
				require.NoError(t, err)

			},
			after: func(t *testing.T) {
			},

			wantCode: http.StatusOK,
			wantResp: test.Result[[]web.Experience]{
				Code: 0,
				Data: []web.Experience{
					web.Experience{
						Id:    2,
						Start: time.Date(2020, 2, 1, 0, 0, 0, 0, time.UTC),
						End:   time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
						Title: "测试工程师",
						Responsibilities: []web.Responsibility{
							web.Responsibility{
								Type:    "bbbbbbb",
								Content: "bbbbbbbbfjsdfjlajlfjadfjladjfldjfldjflajdf",
							},
						},
						Accomplishments: []web.Accomplishment{
							web.Accomplishment{
								Type:    "aaaaaaa",
								Content: "jksjfsjfdlksdjflsjdlfdjsfjslfkls",
							},
						},
						Skills: []string{

							"Jerry",
							"Kelly",
						},
					}, {
						Id:    1,
						Start: time.Date(2019, 2, 1, 0, 0, 0, 0, time.UTC),
						End:   time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
						Title: "测试工程师",
						Responsibilities: []web.Responsibility{
							web.Responsibility{
								Type:    "bbbbbbb",
								Content: "bbbbbbbbfjsdfjlajlfjadfjladjfldjfldjflajdf",
							},
						},
						Accomplishments: []web.Accomplishment{
							{
								Type:    "aaaaaaa",
								Content: "jksjfsjfdlksdjflsjdlfdjsfjslfkls",
							},
						},
						Skills: []string{
							"Jerry",
							"Kelly",
						},
					},
				},
			},
		}, {
			name: "工作经历有重叠",
			before: func(t *testing.T) {
				experiences := []dao.Experience{
					dao.Experience{
						Uid:       uid,
						StartTime: time.Date(2019, 2, 1, 0, 0, 0, 0, time.UTC).UnixMilli(),
						EndTime:   time.Date(2021, 3, 1, 0, 0, 0, 0, time.UTC).UnixMilli(),
						Title:     "测试工程师",
						Responsibilities: sqlx.JsonColumn[[]domain.Responsibility]{
							Valid: true,
							Val: []domain.Responsibility{
								domain.Responsibility{
									Type:    "bbbbbbb",
									Content: "bbbbbbbbfjsdfjlajlfjadfjladjfldjfldjflajdf",
								},
							},
						},
						Accomplishments: sqlx.JsonColumn[[]domain.Accomplishment]{
							Valid: true,
							Val: []domain.Accomplishment{
								domain.Accomplishment{
									Type:    "aaaaaaa",
									Content: "jksjfsjfdlksdjflsjdlfdjsfjslfkls",
								},
							},
						},
						Skills: sqlx.JsonColumn[[]string]{
							Valid: true,
							Val: []string{
								"Jerry",
								"Kelly",
							},
						},
					},
					{
						Uid:       uid,
						StartTime: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli(),
						EndTime:   time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli(),
						Title:     "测试工程师",
						Responsibilities: sqlx.JsonColumn[[]domain.Responsibility]{
							Valid: true,
							Val: []domain.Responsibility{
								domain.Responsibility{
									Type:    "bbbbbbb",
									Content: "bbbbbbbbfjsdfjlajlfjadfjladjfldjfldjflajdf",
								},
							},
						},
						Accomplishments: sqlx.JsonColumn[[]domain.Accomplishment]{
							Valid: true,
							Val: []domain.Accomplishment{
								domain.Accomplishment{
									Type:    "aaaaaaa",
									Content: "jksjfsjfdlksdjflsjdlfdjsfjslfkls",
								},
							},
						},
						Skills: sqlx.JsonColumn[[]string]{
							Valid: true,
							Val: []string{
								"Jerry",
								"Kelly",
							},
						},
					},
				}
				err := s.db.Create(experiences).Error
				require.NoError(t, err)

			},
			after: func(t *testing.T) {
			},

			wantCode: http.StatusOK,
			wantResp: test.Result[[]web.Experience]{
				Code: 0,
				Msg:  "第1段工作经历和第2段工作经历有重合，请提前准备好工作经历重合的理由",
				Data: []web.Experience{
					web.Experience{
						Id:    2,
						Start: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
						End:   time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
						Title: "测试工程师",
						Responsibilities: []web.Responsibility{
							web.Responsibility{
								Type:    "bbbbbbb",
								Content: "bbbbbbbbfjsdfjlajlfjadfjladjfldjfldjflajdf",
							},
						},
						Accomplishments: []web.Accomplishment{
							web.Accomplishment{
								Type:    "aaaaaaa",
								Content: "jksjfsjfdlksdjflsjdlfdjsfjslfkls",
							},
						},
						Skills: []string{

							"Jerry",
							"Kelly",
						},
					}, {
						Id:    1,
						Start: time.Date(2019, 2, 1, 0, 0, 0, 0, time.UTC),
						End:   time.Date(2021, 3, 1, 0, 0, 0, 0, time.UTC),
						Title: "测试工程师",
						Responsibilities: []web.Responsibility{
							web.Responsibility{
								Type:    "bbbbbbb",
								Content: "bbbbbbbbfjsdfjlajlfjadfjladjfldjfldjflajdf",
							},
						},
						Accomplishments: []web.Accomplishment{
							{
								Type:    "aaaaaaa",
								Content: "jksjfsjfdlksdjflsjdlfdjsfjslfkls",
							},
						},
						Skills: []string{
							"Jerry",
							"Kelly",
						},
					},
				},
			},
		}, {
			name: "两段工作经历之间gap太大",
			before: func(t *testing.T) {
				experiences := []dao.Experience{
					dao.Experience{
						Uid:       uid,
						StartTime: time.Date(2019, 2, 1, 0, 0, 0, 0, time.UTC).UnixMilli(),
						EndTime:   time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli(),
						Title:     "测试工程师",
						Responsibilities: sqlx.JsonColumn[[]domain.Responsibility]{
							Valid: true,
							Val: []domain.Responsibility{
								domain.Responsibility{
									Type:    "bbbbbbb",
									Content: "bbbbbbbbfjsdfjlajlfjadfjladjfldjfldjflajdf",
								},
							},
						},
						Accomplishments: sqlx.JsonColumn[[]domain.Accomplishment]{
							Valid: true,
							Val: []domain.Accomplishment{
								domain.Accomplishment{
									Type:    "aaaaaaa",
									Content: "jksjfsjfdlksdjflsjdlfdjsfjslfkls",
								},
							},
						},
						Skills: sqlx.JsonColumn[[]string]{
							Valid: true,
							Val: []string{
								"Jerry",
								"Kelly",
							},
						},
					},
					{
						Uid:       uid,
						StartTime: time.Date(2022, 2, 1, 0, 0, 0, 0, time.UTC).UnixMilli(),
						EndTime:   time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli(),
						Title:     "测试工程师",
						Responsibilities: sqlx.JsonColumn[[]domain.Responsibility]{
							Valid: true,
							Val: []domain.Responsibility{
								domain.Responsibility{
									Type:    "bbbbbbb",
									Content: "bbbbbbbbfjsdfjlajlfjadfjladjfldjfldjflajdf",
								},
							},
						},
						Accomplishments: sqlx.JsonColumn[[]domain.Accomplishment]{
							Valid: true,
							Val: []domain.Accomplishment{
								domain.Accomplishment{
									Type:    "aaaaaaa",
									Content: "jksjfsjfdlksdjflsjdlfdjsfjslfkls",
								},
							},
						},
						Skills: sqlx.JsonColumn[[]string]{
							Valid: true,
							Val: []string{
								"Jerry",
								"Kelly",
							},
						},
					},
				}
				err := s.db.Create(experiences).Error
				require.NoError(t, err)

			},
			after: func(t *testing.T) {
			},

			wantCode: http.StatusOK,
			wantResp: test.Result[[]web.Experience]{
				Code: 0,
				Msg:  "第1段工作经历和第2段工作经历有超过半年的空白期，请提前准备合理的理由",
				Data: []web.Experience{
					web.Experience{
						Id:    2,
						Start: time.Date(2022, 2, 1, 0, 0, 0, 0, time.UTC),
						End:   time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
						Title: "测试工程师",
						Responsibilities: []web.Responsibility{
							web.Responsibility{
								Type:    "bbbbbbb",
								Content: "bbbbbbbbfjsdfjlajlfjadfjladjfldjfldjflajdf",
							},
						},
						Accomplishments: []web.Accomplishment{
							web.Accomplishment{
								Type:    "aaaaaaa",
								Content: "jksjfsjfdlksdjflsjdlfdjsfjslfkls",
							},
						},
						Skills: []string{

							"Jerry",
							"Kelly",
						},
					}, {
						Id:    1,
						Start: time.Date(2019, 2, 1, 0, 0, 0, 0, time.UTC),
						End:   time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
						Title: "测试工程师",
						Responsibilities: []web.Responsibility{
							web.Responsibility{
								Type:    "bbbbbbb",
								Content: "bbbbbbbbfjsdfjlajlfjadfjladjfldjfldjflajdf",
							},
						},
						Accomplishments: []web.Accomplishment{
							{
								Type:    "aaaaaaa",
								Content: "jksjfsjfdlksdjflsjdlfdjsfjslfkls",
							},
						},
						Skills: []string{
							"Jerry",
							"Kelly",
						},
					},
				},
			},
		},
	}
	for _, tc := range testCase {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/resume/experience/list", nil)
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[[]web.Experience]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
			tc.after(t)
			err = s.db.Exec("TRUNCATE TABLE `experiences`").Error
			require.NoError(s.T(), err)

		})
	}
}

func TestExperienceModule(t *testing.T) {
	suite.Run(t, new(ExperienceTestSuite))
}
