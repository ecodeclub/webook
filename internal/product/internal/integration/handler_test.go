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
	"encoding/json"
	"fmt"
	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/product/internal/event"
	"net/http"
	"testing"
	"time"

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

func TestProductModuleTestSuite(t *testing.T) {
	suite.Run(t, new(ProductModuleTestSuite))
}

type ProductModuleTestSuite struct {
	suite.Suite
	server *egin.Component
	db     *egorm.Component
	dao    dao.ProductDAO
	producer mq.Producer
	svc    service.Service
}

func (s *ProductModuleTestSuite) SetupSuite() {
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
	testmq := testioc.InitMQ()
	producer, err := testmq.Producer(event.CreateProductTopic)
	require.NoError(s.T(), err)
	s.producer = producer
}

func (s *ProductModuleTestSuite) TearDownSuite() {
	err := s.db.Exec("DROP TABLE `spus`").Error
	s.NoError(err)
	err = s.db.Exec("DROP TABLE `skus`").Error
	s.NoError(err)
}

func (s *ProductModuleTestSuite) TearDownTest() {
	err := s.db.Exec("TRUNCATE TABLE `spus`").Error
	s.NoError(err)
	err = s.db.Exec("TRUNCATE TABLE `skus`").Error
	s.NoError(err)
}

func (s *ProductModuleTestSuite) TestHandler_RetrieveSKUDetail() {

	t := s.T()

	testCases := []struct {
		name   string
		before func(t *testing.T)

		req      web.SNReq
		wantCode int
		wantResp test.Result[web.SKU]
	}{
		{
			name: "查找成功",
			before: func(t *testing.T) {
				t.Helper()
				spu := dao.SPU{
					Category0:   "product",
					Category1:   "member",
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
			req:      web.SNReq{SN: "SKU001"},
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
		tc := tc
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

func (s *ProductModuleTestSuite) TestHandler_RetrieveSKUDetailFailed() {
	t := s.T()
	testCases := []struct {
		name   string
		before func(t *testing.T)

		req      web.SNReq
		wantCode int
		wantResp test.Result[any]
	}{
		{
			name:     "SN不存在",
			before:   func(t *testing.T) {},
			req:      web.SNReq{SN: "SKU000"},
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
					Category0:   "product",
					Category1:   "member001",
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
			req:      web.SNReq{SN: "SKU002"},
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
					Category0:   "product",
					Category1:   "member002",
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
			req:      web.SNReq{SN: "SKU004"},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
	}
	for _, tc := range testCases {
		tc := tc
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

func (s *ProductModuleTestSuite) TestHandler_RetrieveSPUDetail() {

	t := s.T()

	testCases := []struct {
		name   string
		before func(t *testing.T)

		req      web.SNReq
		wantCode int
		wantResp test.Result[web.SPU]
	}{
		{
			name: "查找成功",
			before: func(t *testing.T) {
				t.Helper()

				spu := dao.SPU{
					Category0:   "product",
					Category1:   "member",
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
			req:      web.SNReq{SN: "SPU102"},
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
		tc := tc
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

func (s *ProductModuleTestSuite) TestHandler_RetrieveSPUDetailFailed() {
	t := s.T()

	testCases := []struct {
		name   string
		before func(t *testing.T)

		req      web.SNReq
		wantCode int
		wantResp test.Result[any]
	}{
		{
			name:     "SN不存在",
			before:   func(t *testing.T) {},
			req:      web.SNReq{SN: "SPU000"},
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
					Category0:   "product",
					Category1:   "member003",
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
			req:      web.SNReq{SN: "SPU103"},
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
					Category0:   "product",
					Category1:   "member005",
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
			req:      web.SNReq{SN: "SPU104"},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
	}
	for _, tc := range testCases {
		tc := tc
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

func (s *ProductModuleTestSuite) TestService_FindSPUByID() {
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
				t.Helper()
				spu := dao.SPU{
					Category0:   "product",
					Category1:   "member006",
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
				SN:        "SPU1102",
				Name:      "会员服务-2",
				Category0: "product",
				Category1: "member006",
				Desc:      "提供不同期限的会员服务-2",
				Status:    domain.StatusOnShelf,
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
		tc := tc
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

func (s *ProductModuleTestSuite) TestService_Save() {
	t := s.T()
	testCases := []struct {
		name     string
		before   func(t *testing.T)
		req      web.SPUSaveReq
		after    func(t *testing.T)
		wantCode int
		wantResp test.Result[int64]
	}{
		{
			name:   "新增",
			before: func(t *testing.T) {},
			after: func(t *testing.T) {
				spu, err := s.dao.FindSPUByID(context.Background(), 1)
				require.NoError(t, err)
				skus, err := s.dao.FindSKUsBySPUID(context.Background(), 1)
				require.NoError(t, err)
				s.assertSpu(t, dao.SPU{
					Category0:   "code",
					Category1:   "project",
					Name:        "project1",
					Description: "projectDesc",
					Status:      2,
				}, spu)
				s.assertSkus(t, []dao.SKU{
					{
						SPUID:       spu.Id,
						Name:        "skuName2",
						Description: "skuDesc2",
						Price:       100,
						Stock:       1,
						StockLimit:  100,
						SaleType:    1,
						Attrs: sql.NullString{
							Valid:  true,
							String: `{"id":2}`,
						},
						Image:  "image.com",
						Status: 2,
					},
					{
						SPUID:       spu.Id,
						Name:        "skuName1",
						Description: "skuDesc1",
						Price:       100,
						Stock:       1,
						StockLimit:  100,
						SaleType:    1,
						Attrs: sql.NullString{
							Valid:  true,
							String: `{"id":1}`,
						},
						Image:  "image.com",
						Status: 2,
					},
				}, skus)

			},
			req: web.SPUSaveReq{
				SPU: web.SPU{
					Name: "project1",
					Desc: "projectDesc",
					Category0: web.Category{
						Name: "code",
					},
					Category1: web.Category{
						Name: "project",
					},
					SKUs: []web.SKU{
						{
							Name:       "skuName1",
							Desc:       "skuDesc1",
							Price:      100,
							Stock:      1,
							StockLimit: 100,
							SaleType:   1,
							Attrs:      `{"id":1}`,
							Image:      "image.com",
						},
						{
							Name:       "skuName2",
							Desc:       "skuDesc2",
							Price:      100,
							Stock:      1,
							StockLimit: 100,
							SaleType:   1,
							Attrs:      `{"id":2}`,
							Image:      "image.com",
						},
					},
				},
			},
			wantCode: 200,
			wantResp: test.Result[int64]{
				Data: 1,
			},
		},
		{
			name: "编辑",
			before: func(t *testing.T) {
				_, err := s.dao.SaveProduct(context.Background(),
					dao.SPU{
						Id:          1,
						Category0:   "product",
						Category1:   "project",
						SN:          "skn1",
						Name:        "code1",
						Description: "sknDesc",
						Status:      2,
						Ctime:       1,
						Utime:       2,
					}, []dao.SKU{
						{
							Id:          1,
							SPUID:       1,
							Name:        "skuName1",
							SN:          "spu1",
							Description: "skuDesc1",
							Price:       99,
							Stock:       1,
							StockLimit:  10,
							SaleType:    1,
							Attrs: sql.NullString{
								Valid:  true,
								String: "1",
							},
							Image:  "image.com",
							Status: 2,
							Ctime:  1,
							Utime:  2,
						},
						{
							Id:          2,
							SPUID:       1,
							SN:          "spu2",
							Name:        "skuName2",
							Description: "skuDesc2",
							Price:       98,
							Stock:       2,
							StockLimit:  19,
							SaleType:    1,
							Attrs: sql.NullString{
								Valid:  true,
								String: "1",
							},
							Image:  "image.com",
							Status: 2,
							Ctime:  1,
							Utime:  2,
						},
					})
				require.NoError(t, err)

			},
			wantCode: 200,
			wantResp: test.Result[int64]{
				Data: 1,
			},
			after: func(t *testing.T) {
				spu, err := s.dao.FindSPUByID(context.Background(), 1)
				require.NoError(t, err)
				skus, err := s.dao.FindSKUsBySPUID(context.Background(), 1)
				require.NoError(t, err)
				s.assertSpu(t, dao.SPU{
					Category0:   "newProduct",
					Category1:   "newProject",
					Name:        "newCode1",
					Description: "newSknDesc",
					Status:      2,
				}, spu)
				s.assertSkus(t, []dao.SKU{
					{
						SPUID:       1,
						Name:        "newSku1",
						Description: "newSkuDesc1",
						Price:       98,
						Stock:       2,
						StockLimit:  11,
						SaleType:    2,
						Attrs: sql.NullString{
							Valid:  true,
							String: "2",
						},
						Image:  "image.com.new",
						Status: 2,
					},
					{
						SPUID:       1,
						Name:        "skuName3",
						Description: "skuDesc3",
						Price:       90,
						Stock:       8,
						StockLimit:  8,
						SaleType:    2,
						Attrs: sql.NullString{
							Valid:  true,
							String: "33",
						},
						Image:  "image3.com",
						Status: 2,
					},
				}, skus)
			},
			req: web.SPUSaveReq{
				SPU: web.SPU{
					ID:   1,
					SN:   "skn1",
					Name: "newCode1",
					Desc: "newSknDesc",
					Category0: web.Category{
						Name: "newProduct",
					},
					Category1: web.Category{
						Name: "newProject",
					},
					SKUs: []web.SKU{
						{
							ID:         1,
							Name:       "newSku1",
							Desc:       "newSkuDesc1",
							Price:      98,
							Stock:      2,
							StockLimit: 11,
							SaleType:   2,
							Attrs:      "2",
							Image:      "image.com.new",
						},
						{
							Name:       "skuName3",
							Desc:       "skuDesc3",
							Price:      90,
							Stock:      8,
							StockLimit: 8,
							SaleType:   2,
							Attrs:      "33",
							Image:      "image3.com",
						},
					},
				},
			},
		},
		{
			name: "sn相同编辑",
			before: func(t *testing.T) {
				_, err := s.dao.SaveProduct(context.Background(),
					dao.SPU{
						Id:          1,
						Category0:   "product",
						Category1:   "project",
						SN:          "spn1",
						Name:        "code1",
						Description: "sknDesc",
						Status:      2,
						Ctime:       1,
						Utime:       2,
					}, []dao.SKU{
						{
							Id:          1,
							SPUID:       1,
							Name:        "skuName1",
							SN:          "sku1",
							Description: "skuDesc1",
							Price:       99,
							Stock:       1,
							StockLimit:  10,
							SaleType:    1,
							Attrs: sql.NullString{
								Valid:  true,
								String: "1",
							},
							Image:  "image.com",
							Status: 2,
							Ctime:  1,
							Utime:  2,
						},
						{
							Id:          2,
							SPUID:       1,
							SN:          "sku2",
							Name:        "skuName2",
							Description: "skuDesc2",
							Price:       98,
							Stock:       2,
							StockLimit:  19,
							SaleType:    1,
							Attrs: sql.NullString{
								Valid:  true,
								String: "1",
							},
							Image:  "image.com",
							Status: 2,
							Ctime:  1,
							Utime:  2,
						},
					})
				require.NoError(t, err)

			},
			wantCode: 200,
			wantResp: test.Result[int64]{
				Data: 1,
			},
			after: func(t *testing.T) {
				spu, err := s.dao.FindSPUByID(context.Background(), 1)
				require.NoError(t, err)
				skus, err := s.dao.FindSKUsBySPUID(context.Background(), 1)
				require.NoError(t, err)
				s.assertSpu(t, dao.SPU{
					Category0:   "newProduct",
					Category1:   "newProject",
					Name:        "newCode1",
					Description: "newSknDesc",
					Status:      2,
				}, spu)
				s.assertSkus(t, []dao.SKU{
					{
						SPUID:       1,
						Name:        "newSku1",
						Description: "newSkuDesc1",
						Price:       98,
						Stock:       2,
						StockLimit:  11,
						SaleType:    2,
						Attrs: sql.NullString{
							Valid:  true,
							String: "2",
						},
						Image:  "image.com.new",
						Status: 2,
					},
					{
						SPUID:       1,
						Name:        "skuName3",
						Description: "skuDesc3",
						Price:       90,
						Stock:       8,
						StockLimit:  8,
						SaleType:    2,
						Attrs: sql.NullString{
							Valid:  true,
							String: "33",
						},
						Image:  "image3.com",
						Status: 2,
					},
				}, skus)
			},
			req: web.SPUSaveReq{
				SPU: web.SPU{
					SN:   "spn1",
					Name: "newCode1",
					Desc: "newSknDesc",
					Category0: web.Category{
						Name: "newProduct",
					},
					Category1: web.Category{
						Name: "newProject",
					},
					SKUs: []web.SKU{
						{
							SN:         "sku1",
							Name:       "newSku1",
							Desc:       "newSkuDesc1",
							Price:      98,
							Stock:      2,
							StockLimit: 11,
							SaleType:   2,
							Attrs:      "2",
							Image:      "image.com.new",
						},
						{
							SN:         "sku3",
							Name:       "skuName3",
							Desc:       "skuDesc3",
							Price:      90,
							Stock:      8,
							StockLimit: 8,
							SaleType:   2,
							Attrs:      "33",
							Image:      "image3.com",
						},
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/product/save", iox.NewJSONReader(tc.req))
			require.NoError(t, err)
			req.Header.Set("content-type", "application/json")
			recorder := test.NewJSONResponseRecorder[int64]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
			tc.after(t)
			err = s.db.Exec("TRUNCATE TABLE `spus`").Error
			s.NoError(err)
			err = s.db.Exec("TRUNCATE TABLE `skus`").Error
			s.NoError(err)
		})
	}
}

func (s *ProductModuleTestSuite) TestService_Event() {
	testcases := []struct {
		name  string
		evt   event.SPUEvent
		after func(t *testing.T)
	}{
		{
			name: "同步商品信息",
			evt: event.SPUEvent{
				UID:       123,
				ID:        1,
				SN:        "spu1",
				Name:      "project1",
				Desc:      "desc1",
				Category1: "category1",
				Category0: "category0",
				SKUs: []event.SKU{
					{
						SN:         "skusn",
						Name:       "skuName",
						Desc:       "description",
						Price:      1,
						Stock:      1,
						StockLimit: 100,
						SaleType:   domain.SaleTypeUnlimited.ToUint8(),
						Attrs:      "11",
						Image:      "image.com",
					},
				},
			},
			after: func(t *testing.T) {
				spu, err := s.dao.FindSPUByID(context.Background(), 1)
				require.NoError(t, err)
				skus, err := s.dao.FindSKUsBySPUID(context.Background(), 1)
				require.NoError(t, err)
				s.assertSpu(t, dao.SPU{
					Category0:   "category0",
					Category1:   "category1",
					Name:        "project1",
					Description: "desc1",
					Status:      2,
				}, spu)
				s.assertSkus(t, []dao.SKU{
					{
						SPUID:       spu.Id,
						Name:        "skuName",
						Description: "description",
						Price:       1,
						Stock:       1,
						StockLimit:  100,
						SaleType:    domain.SaleTypeUnlimited.ToUint8(),
						Attrs: sql.NullString{
							Valid:  true,
							String: "11",
						},
						Image:  "image.com",
						Status: 2,
					},
				}, skus)
			},
		},
	}
	for _, tc := range testcases {
		s.T().Run(tc.name, func(t *testing.T) {
			evtJson,err := json.Marshal(tc.evt)
			require.NoError(t, err)
			_,err = s.producer.Produce(context.Background(),&mq.Message{
				Value: evtJson,
			})
			require.NoError(t, err)
			time.Sleep(5*time.Second)
		})
	}

}

func (s *ProductModuleTestSuite) TestService_List() {
	t := s.T()
	testCases := []struct {
		name     string
		req      web.SPUListReq
		before   func(t *testing.T)
		wantResp web.SPUListResp
		wantCode int
	}{
		{
			name: "列表",
			req: web.SPUListReq{
				Limit:  2,
				Offset: 0,
			},
			before: func(t *testing.T) {
				s.genSPUs(100, 2)
			},
			wantCode: 200,
			wantResp: web.SPUListResp{
				List: []web.SPU{
					{
						ID:   100,
						SN:   "100",
						Name: "name100",
						Desc: "desc100",
						Category0: web.Category{
							Name: "category0100",
						},
						Category1: web.Category{
							Name: "category1100",
						},
					},
					{
						ID:   99,
						SN:   "99",
						Name: "name99",
						Desc: "desc99",
						Category0: web.Category{
							Name: "category099",
						},
						Category1: web.Category{
							Name: "category199",
						},
					},
				},
				Count: 100,
			},
		},
		{
			name: "分页",
			req: web.SPUListReq{
				Limit:  2,
				Offset: 10,
			},
			before: func(t *testing.T) {
				s.genSPUs(100, 2)
			},
			wantCode: 200,
			wantResp: web.SPUListResp{
				List: []web.SPU{
					{
						ID:   90,
						SN:   "90",
						Name: "name90",
						Desc: "desc90",
						Category0: web.Category{
							Name: "category090",
						},
						Category1: web.Category{
							Name: "category190",
						},
					},

					{
						ID:   89,
						SN:   "89",
						Name: "name89",
						Desc: "desc89",
						Category0: web.Category{
							Name: "category089",
						},
						Category1: web.Category{
							Name: "category189",
						},
					},
				},
				Count: 100,
			},
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/product/spu/list", iox.NewJSONReader(tc.req))
			require.NoError(t, err)
			req.Header.Set("content-type", "application/json")
			recorder := test.NewJSONResponseRecorder[web.SPUListResp]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			resp := recorder.MustScan()
			assert.Equal(t, tc.wantResp, resp.Data)
			err = s.db.Exec("TRUNCATE TABLE `spus`").Error
			s.NoError(err)
			err = s.db.Exec("TRUNCATE TABLE `skus`").Error
			s.NoError(err)
		})
	}
}

func (s *ProductModuleTestSuite) assertSpu(t *testing.T, wantSpu dao.SPU, actualSpu dao.SPU) {
	assert.True(t, actualSpu.Ctime != 0)
	assert.True(t, actualSpu.Utime != 0)
	assert.True(t, actualSpu.Id != 0)
	assert.True(t, actualSpu.SN != "")
	actualSpu.Ctime = 0
	actualSpu.Utime = 0
	actualSpu.Id = 0
	actualSpu.SN = ""
	assert.Equal(t, wantSpu, actualSpu)
}

func (s *ProductModuleTestSuite) assertSkus(t *testing.T, wantSkus []dao.SKU, actualSkus []dao.SKU) {
	for idx := range actualSkus {
		assert.True(t, actualSkus[idx].Ctime != 0)
		assert.True(t, actualSkus[idx].Utime != 0)
		assert.True(t, actualSkus[idx].Id != 0)
		assert.True(t, actualSkus[idx].SN != "")
		actualSkus[idx].Ctime = 0
		actualSkus[idx].Utime = 0
		actualSkus[idx].Id = 0
		actualSkus[idx].SN = ""
	}
	assert.ElementsMatch(t, wantSkus, actualSkus)
}

func (s *ProductModuleTestSuite) genSPUs(onselfNumber, offselfNumber int) {
	spus := make([]dao.SPU, 0, onselfNumber)
	for i := 1; i <= onselfNumber; i++ {
		spus = append(spus, s.genSPU(int64(i)))
	}
	for i := 1; i <= offselfNumber; i++ {
		spu := s.genSPU(int64(i + onselfNumber))
		spu.Status = domain.StatusOffShelf.ToUint8()
		spus = append(spus, spu)
	}
	err := s.db.WithContext(context.Background()).
		Create(&spus).Error
	require.NoError(s.T(), err)
}

func (s *ProductModuleTestSuite) genSPU(i int64) dao.SPU {
	return dao.SPU{
		Id:          int64(i),
		Category0:   fmt.Sprintf("category0%d", i),
		Category1:   fmt.Sprintf("category1%d", i),
		SN:          fmt.Sprintf("%d", i),
		Name:        fmt.Sprintf("name%d", i),
		Description: fmt.Sprintf("desc%d", i),
		Status:      domain.StatusOnShelf.ToUint8(),
		Ctime:       int64(i),
		Utime:       int64(i),
	}
}