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
	"net/http"
	"testing"

	"github.com/ecodeclub/ekit/iox"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/product/internal/errs"
	"github.com/ecodeclub/webook/internal/product/internal/integration/startup"
	"github.com/ecodeclub/webook/internal/product/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/product/internal/web"
	"github.com/ecodeclub/webook/internal/test"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/ego-component/egorm"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/econf"
	"github.com/gotomicro/ego/server/egin"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const uid = 123

type HandlerTestSuite struct {
	suite.Suite
	server *egin.Component
	db     *egorm.Component
	dao    dao.ProductDAO
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
}

func (s *HandlerTestSuite) TearDownSuite() {
	err := s.db.Exec("DROP TABLE `product_spus`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("DROP TABLE `product_skus`").Error
	require.NoError(s.T(), err)
}

func (s *HandlerTestSuite) TearDownTest() {
	err := s.db.Exec("TRUNCATE TABLE `product_spus`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE `product_skus`").Error
	require.NoError(s.T(), err)
}

func (s *HandlerTestSuite) TestProductDetail() {

	testCases := []struct {
		name   string
		before func(t *testing.T)

		req      web.ProductSNReq
		wantCode int
		wantResp test.Result[web.Product]
	}{
		{
			name: "查找成功",
			before: func(t *testing.T) {
				spus := []dao.ProductSPU{
					{
						SN:          "SPU001",
						Name:        "会员服务",
						Description: "提供不同期限的会员服务",
						Status:      dao.StatusOnShelf,
					},
				}
				for i := 0; i < len(spus); i++ {
					_, err := s.dao.CreateSPU(context.Background(), spus[i])
					require.NoError(t, err)
				}

				skus := []dao.ProductSKU{
					{
						SN:           "SKU001",
						ProductSPUID: 1,
						Name:         "星期会员",
						Description:  "提供一周的会员服务",
						Price:        decimal.RequireFromString("7.99"),
						Stock:        1000,
						StockLimit:   100000000,
						Status:       dao.StatusOnShelf,
					},
				}
				for i := 0; i < len(skus); i++ {
					_, err := s.dao.CreateSKU(context.Background(), skus[i])
					require.NoError(t, err)
				}
			},
			req:      web.ProductSNReq{SN: "SKU001"},
			wantCode: 200,
			wantResp: test.Result[web.Product]{
				Data: web.Product{
					SPU: web.ProductSPU{
						SN:   "SPU001",
						Name: "会员服务",
						Desc: "提供不同期限的会员服务",
					},
					SKU: web.ProductSKU{
						SN:         "SKU001",
						Name:       "星期会员",
						Desc:       "提供一周的会员服务",
						Price:      "7.99",
						Stock:      1000,
						StockLimit: 100000000,
						SaleType:   1,
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/product/detail", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.Product]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
		})
	}
}

func (s *HandlerTestSuite) TestProductDetailFailed() {
	testCases := []struct {
		name   string
		before func(t *testing.T)

		req      web.ProductSNReq
		wantCode int
		wantResp test.Result[any]
	}{
		{
			name:     "SN不存在",
			before:   func(t *testing.T) {},
			req:      web.ProductSNReq{SN: "SKU000"},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
		{
			name: "SPU上架_SKU下架",
			before: func(t *testing.T) {
				spus := []dao.ProductSPU{
					{
						SN:          "SPU001",
						Name:        "会员服务",
						Description: "提供不同期限的会员服务",
						Status:      dao.StatusOnShelf,
					},
				}
				for i := 0; i < len(spus); i++ {
					_, err := s.dao.CreateSPU(context.Background(), spus[i])
					require.NoError(t, err)
				}

				skus := []dao.ProductSKU{
					{
						SN:           "SKU002",
						ProductSPUID: 1,
						Name:         "月会员",
						Description:  "提供一个月的会员服务",
						Price:        decimal.RequireFromString("9.99"),
						Stock:        1000,
						StockLimit:   100000000,
						Status:       dao.StatusOffShelf,
					},
				}
				for i := 0; i < len(skus); i++ {
					_, err := s.dao.CreateSKU(context.Background(), skus[i])
					require.NoError(t, err)
				}
			},
			req:      web.ProductSNReq{SN: "SKU002"},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
		{
			name: "SPU下架_SKU上架",
			before: func(t *testing.T) {
				spus := []dao.ProductSPU{
					{
						SN:          "SPU002",
						Name:        "会员服务",
						Description: "提供不同期限的会员服务",
						Status:      dao.StatusOffShelf,
					},
				}
				for i := 0; i < len(spus); i++ {
					_, err := s.dao.CreateSPU(context.Background(), spus[i])
					require.NoError(t, err)
				}

				skus := []dao.ProductSKU{
					{
						SN:           "SKU003",
						ProductSPUID: 2,
						Name:         "季度会员",
						Description:  "提供一个季度的会员服务",
						Price:        decimal.RequireFromString("29.70"),
						Stock:        1000,
						StockLimit:   100000000,
						Status:       dao.StatusOnShelf,
					},
				}
				for i := 0; i < len(skus); i++ {
					_, err := s.dao.CreateSKU(context.Background(), skus[i])
					require.NoError(t, err)
				}
			},
			req:      web.ProductSNReq{SN: "SKU003"},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
		{
			name: "SPU下架_SKU下架",
			before: func(t *testing.T) {
				spus := []dao.ProductSPU{
					{
						SN:          "SPU003",
						Name:        "会员服务",
						Description: "提供不同期限的会员服务",
						Status:      dao.StatusOffShelf,
					},
				}
				for i := 0; i < len(spus); i++ {
					_, err := s.dao.CreateSPU(context.Background(), spus[i])
					require.NoError(t, err)
				}

				skus := []dao.ProductSKU{
					{
						SN:           "SKU004",
						ProductSPUID: 3,
						Name:         "年会员",
						Description:  "提供一年的会员服务",
						Price:        decimal.RequireFromString("118.80"),
						Stock:        1000,
						StockLimit:   100000000,
						Status:       dao.StatusOffShelf,
					},
				}
				for i := 0; i < len(skus); i++ {
					_, err := s.dao.CreateSKU(context.Background(), skus[i])
					require.NoError(t, err)
				}
			},
			req:      web.ProductSNReq{SN: "SKU004"},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
	}
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/product/detail", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[any]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
		})
	}
}

func TestHandler(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}
