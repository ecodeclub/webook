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
	"net/http"
	"testing"

	"github.com/ecodeclub/ekit/iox"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/product/internal/domain"
	"github.com/ecodeclub/webook/internal/product/internal/errs"
	"github.com/ecodeclub/webook/internal/product/internal/integration/startup"
	"github.com/ecodeclub/webook/internal/product/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/product/internal/service"
	"github.com/ecodeclub/webook/internal/product/internal/web"
	"github.com/ecodeclub/webook/internal/test"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/ego-component/egorm"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/econf"
	"github.com/gotomicro/ego/server/egin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const uid = int64(123)

func TestProductHandler(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}

type HandlerTestSuite struct {
	suite.Suite
	server *egin.Component
	db     *egorm.Component
	dao    dao.ProductDAO
	svc    service.Service
}

func (s *HandlerTestSuite) SetupSuite() {
	handler, err := startup.InitHandler()
	require.NoError(s.T(), err)

	econf.Set("server", map[string]any{"contextTimeout": "1s"})
	server := egin.Load("server").Build()
	server.Use(func(ctx *gin.Context) {
		ctx.Set("_session", session.NewMemorySession(session.Claims{
			Uid: uid,
		}))
	})
	handler.PrivateRoutes(server.Engine)

	s.server = server
	s.db = testioc.InitDB()
	err = dao.InitTables(s.db)
	require.NoError(s.T(), err)
	s.dao = dao.NewProductGORMDAO(s.db)
	s.svc = startup.InitService()
}

func (s *HandlerTestSuite) TearDownSuite() {
	err := s.db.Exec("DROP TABLE `spus`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("DROP TABLE `skus`").Error
	require.NoError(s.T(), err)
}

func (s *HandlerTestSuite) TearDownTest() {
	err := s.db.Exec("TRUNCATE TABLE `spus`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE `skus`").Error
	require.NoError(s.T(), err)
}

func (s *HandlerTestSuite) TestHandler_RetrieveSKUDetail() {

	t := s.T()

	testCases := []struct {
		name   string
		before func(t *testing.T)

		req      web.SPUSNReq
		wantCode int
		wantResp test.Result[web.SKU]
	}{
		{
			name: "查找成功",
			before: func(t *testing.T) {
				spu := dao.SPU{
					SN:          "SPU001",
					Name:        "会员服务",
					Description: "提供不同期限的会员服务",
					Status:      domain.StatusOnShelf.ToUint8(),
				}
				id, err := s.dao.CreateSPU(context.Background(), spu)
				require.NoError(t, err)

				skus := []dao.SKU{
					{
						SN:          "SKU001",
						SPUID:       id,
						Name:        "星期会员",
						Description: "提供一周的会员服务",
						Price:       799,
						Stock:       1000,
						StockLimit:  100000000,
						Status:      domain.StatusOnShelf.ToUint8(),
						Attrs:       sql.NullString{String: `{"days":7}`, Valid: true},
						Image:       "image-SKU001",
					},
				}
				for i := 0; i < len(skus); i++ {
					_, err := s.dao.CreateSKU(context.Background(), skus[i])
					require.NoError(t, err)
				}
			},
			req:      web.SPUSNReq{SN: "SKU001"},
			wantCode: 200,
			wantResp: test.Result[web.SKU]{
				Data: web.SKU{
					SN:         "SKU001",
					Name:       "星期会员",
					Desc:       "提供一周的会员服务",
					Price:      799,
					Stock:      1000,
					StockLimit: 100000000,
					SaleType:   domain.SaleTypeUnlimited.ToUint8(),
					Attrs:      `{"days":7}`,
					Image:      "image-SKU001",
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/product/sku/detail", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.SKU]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
		})
	}
}

func (s *HandlerTestSuite) TestHandler_RetrieveSKUDetailFailed() {
	t := s.T()
	testCases := []struct {
		name   string
		before func(t *testing.T)

		req      web.SPUSNReq
		wantCode int
		wantResp test.Result[any]
	}{
		{
			name:     "SN不存在",
			before:   func(t *testing.T) {},
			req:      web.SPUSNReq{SN: "SKU000"},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
		{
			name: "SPU上架_SKU下架",
			before: func(t *testing.T) {
				t.Helper()
				spu := dao.SPU{
					SN:          "SPU002",
					Name:        "会员服务",
					Description: "提供不同期限的会员服务",
					Status:      domain.StatusOnShelf.ToUint8(),
				}
				id, err := s.dao.CreateSPU(context.Background(), spu)
				require.NoError(t, err)

				skus := []dao.SKU{
					{
						SN:          "SKU002",
						SPUID:       id,
						Name:        "月会员",
						Description: "提供一个月的会员服务",
						Price:       999,
						Stock:       1000,
						StockLimit:  100000000,
						Status:      domain.StatusOffShelf.ToUint8(),
						Attrs:       sql.NullString{String: `{"days":31}`, Valid: true},
						Image:       "image-SKU002",
					},
				}
				for i := 0; i < len(skus); i++ {
					_, err := s.dao.CreateSKU(context.Background(), skus[i])
					require.NoError(t, err)
				}
			},
			req:      web.SPUSNReq{SN: "SKU002"},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
		{
			name: "SPU下架_SKU下架",
			before: func(t *testing.T) {
				t.Helper()
				spu := dao.SPU{
					SN:          "SPU004",
					Name:        "会员服务",
					Description: "提供不同期限的会员服务",
					Status:      domain.StatusOffShelf.ToUint8(),
				}

				id, err := s.dao.CreateSPU(context.Background(), spu)
				require.NoError(t, err)

				skus := []dao.SKU{
					{
						SN:          "SKU004",
						SPUID:       id,
						Name:        "年会员",
						Description: "提供一年的会员服务",
						Price:       11880,
						Stock:       1000,
						StockLimit:  100000000,
						Status:      domain.StatusOffShelf.ToUint8(),
						Attrs:       sql.NullString{String: `{"days":366}`, Valid: true},
						Image:       "image-SKU004",
					},
				}
				for i := 0; i < len(skus); i++ {
					_, err := s.dao.CreateSKU(context.Background(), skus[i])
					require.NoError(t, err)
				}
			},
			req:      web.SPUSNReq{SN: "SKU004"},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/product/sku/detail", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[any]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
		})
	}
}

func (s *HandlerTestSuite) TestHandler_RetrieveSPUDetail() {

	t := s.T()

	testCases := []struct {
		name   string
		before func(t *testing.T)

		req      web.SPUSNReq
		wantCode int
		wantResp test.Result[web.SPU]
	}{
		{
			name: "查找成功",
			before: func(t *testing.T) {
				spu := dao.SPU{
					SN:          "SPU102",
					Name:        "会员服务-2",
					Description: "提供不同期限的会员服务-2",
					Status:      domain.StatusOnShelf.ToUint8(),
				}
				id, err := s.dao.CreateSPU(context.Background(), spu)
				require.NoError(t, err)

				skus := []dao.SKU{
					{
						SN:          "SKU101",
						SPUID:       id,
						Name:        "月会员",
						Description: "提供一个月会员服务",
						Price:       1899,
						Stock:       1000,
						StockLimit:  100000000,
						Status:      domain.StatusOnShelf.ToUint8(),
						Attrs:       sql.NullString{String: `{"days":31}`, Valid: true},
						Image:       "image-SKU101",
					},
					{
						SN:          "SKU102",
						SPUID:       id,
						Name:        "星期会员",
						Description: "提供一周的会员服务",
						Price:       799,
						Stock:       1000,
						StockLimit:  100000000,
						Status:      domain.StatusOnShelf.ToUint8(),
						Attrs:       sql.NullString{String: `{"days":7}`, Valid: true},
						Image:       "image-SKU102",
					},
				}
				for i := 0; i < len(skus); i++ {
					_, err := s.dao.CreateSKU(context.Background(), skus[i])
					require.NoError(t, err)
				}
			},
			req:      web.SPUSNReq{SN: "SPU102"},
			wantCode: 200,
			wantResp: test.Result[web.SPU]{
				Data: web.SPU{
					SN:   "SPU102",
					Name: "会员服务-2",
					Desc: "提供不同期限的会员服务-2",
					SKUs: []web.SKU{
						{
							SN:         "SKU102",
							Name:       "星期会员",
							Desc:       "提供一周的会员服务",
							Price:      799,
							Stock:      1000,
							StockLimit: 100000000,
							SaleType:   domain.SaleTypeUnlimited.ToUint8(),
							Attrs:      `{"days":7}`,
							Image:      "image-SKU102",
						},
						{
							SN:         "SKU101",
							Name:       "月会员",
							Desc:       "提供一个月会员服务",
							Price:      1899,
							Stock:      1000,
							StockLimit: 100000000,
							SaleType:   domain.SaleTypeUnlimited.ToUint8(),
							Attrs:      `{"days":31}`,
							Image:      "image-SKU101",
						},
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/product/spu/detail", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.SPU]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
		})
	}
}

func (s *HandlerTestSuite) TestHandler_RetrieveSPUDetailFailed() {
	t := s.T()
	testCases := []struct {
		name   string
		before func(t *testing.T)

		req      web.SPUSNReq
		wantCode int
		wantResp test.Result[any]
	}{
		{
			name:     "SN不存在",
			before:   func(t *testing.T) {},
			req:      web.SPUSNReq{SN: "SPU000"},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
		{
			name: "SPU下架_SKU上架",
			before: func(t *testing.T) {
				t.Helper()
				spu := dao.SPU{
					SN:          "SPU103",
					Name:        "会员服务-3",
					Description: "提供不同期限的会员服务-3",
					Status:      domain.StatusOffShelf.ToUint8(),
				}
				id, err := s.dao.CreateSPU(context.Background(), spu)
				require.NoError(t, err)

				skus := []dao.SKU{
					{
						SN:          "SKU103",
						SPUID:       id,
						Name:        "季度会员",
						Description: "提供一个季度的会员服务",
						Price:       2970,
						Stock:       1000,
						StockLimit:  100000000,
						Status:      domain.StatusOnShelf.ToUint8(),
						Attrs:       sql.NullString{String: `{"days":100}`, Valid: true},
						Image:       "image-SKU003",
					},
				}
				for i := 0; i < len(skus); i++ {
					_, err := s.dao.CreateSKU(context.Background(), skus[i])
					require.NoError(t, err)
				}
			},
			req:      web.SPUSNReq{SN: "SPU103"},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
		{
			name: "SPU下架_SKU下架",
			before: func(t *testing.T) {
				t.Helper()
				spu := dao.SPU{
					SN:          "SPU104",
					Name:        "会员服务-4",
					Description: "提供不同期限的会员服务-4",
					Status:      domain.StatusOffShelf.ToUint8(),
				}

				id, err := s.dao.CreateSPU(context.Background(), spu)
				require.NoError(t, err)

				skus := []dao.SKU{
					{
						SN:          "SKU104",
						SPUID:       id,
						Name:        "年会员",
						Description: "提供一年的会员服务",
						Price:       11880,
						Stock:       1000,
						StockLimit:  100000000,
						Status:      domain.StatusOffShelf.ToUint8(),
						Attrs:       sql.NullString{String: `{"days":366}`, Valid: true},
						Image:       "image-SKU004",
					},
				}
				for i := 0; i < len(skus); i++ {
					_, err := s.dao.CreateSKU(context.Background(), skus[i])
					require.NoError(t, err)
				}
			},
			req:      web.SPUSNReq{SN: "SPU104"},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/product/spu/detail", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[any]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
		})
	}
}

func (s *HandlerTestSuite) TestService_FindSPUByID() {
	t := s.T()
	testCases := []struct {
		name     string
		getSPUID func(t *testing.T) int64

		SPU           domain.SPU
		errRequreFunc require.ErrorAssertionFunc
	}{
		{
			name: "查找成功",
			getSPUID: func(t *testing.T) int64 {
				spu := dao.SPU{
					SN:          "SPU1102",
					Name:        "会员服务-2",
					Description: "提供不同期限的会员服务-2",
					Status:      domain.StatusOnShelf.ToUint8(),
				}
				id, err := s.dao.CreateSPU(context.Background(), spu)
				require.NoError(t, err)

				skus := []dao.SKU{
					{
						SN:          "SKU1101",
						SPUID:       id,
						Name:        "月会员",
						Description: "提供一个月会员服务",
						Price:       1899,
						Stock:       1000,
						StockLimit:  100000000,
						Status:      domain.StatusOnShelf.ToUint8(),
						Attrs:       sql.NullString{String: `{"days":31}`, Valid: true},
						Image:       "image-SKU1101",
					},
					{
						SN:          "SKU1102",
						SPUID:       id,
						Name:        "星期会员",
						Description: "提供一周的会员服务",
						Price:       799,
						Stock:       1000,
						StockLimit:  100000000,
						Status:      domain.StatusOnShelf.ToUint8(),
						Attrs:       sql.NullString{String: `{"days":7}`, Valid: true},
						Image:       "image-SKU1102",
					},
				}
				for i := 0; i < len(skus); i++ {
					_, err := s.dao.CreateSKU(context.Background(), skus[i])
					require.NoError(t, err)
				}
				return id
			},
			SPU: domain.SPU{
				SN:     "SPU1102",
				Name:   "会员服务-2",
				Desc:   "提供不同期限的会员服务-2",
				Status: domain.StatusOnShelf,
				SKUs: []domain.SKU{
					{
						SN:         "SKU1102",
						Name:       "星期会员",
						Desc:       "提供一周的会员服务",
						Price:      799,
						Stock:      1000,
						StockLimit: 100000000,
						SaleType:   domain.SaleTypeUnlimited,
						Attrs:      `{"days":7}`,
						Image:      "image-SKU1102",
						Status:     domain.StatusOnShelf,
					},
					{
						SN:         "SKU1101",
						Name:       "月会员",
						Desc:       "提供一个月会员服务",
						Price:      1899,
						Stock:      1000,
						StockLimit: 100000000,
						SaleType:   domain.SaleTypeUnlimited,
						Attrs:      `{"days":31}`,
						Image:      "image-SKU1101",
						Status:     domain.StatusOnShelf,
					},
				},
			},
			errRequreFunc: require.NoError,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			id := tc.getSPUID(t)
			spu, err := s.svc.FindSPUByID(context.Background(), id)
			tc.errRequreFunc(t, err)
			if err == nil {
				require.NotZero(t, spu.ID)
				spu.ID = 0
				for i := 0; i < len(spu.SKUs); i++ {
					require.NotZero(t, spu.SKUs[i].ID)
					require.NotZero(t, spu.SKUs[i].SPUID)
					spu.SKUs[i].ID = 0
					spu.SKUs[i].SPUID = 0
				}
				require.Equal(t, tc.SPU, spu)
			}
		})
	}
}
