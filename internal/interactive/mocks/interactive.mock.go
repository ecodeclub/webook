// Code generated by MockGen. DO NOT EDIT.
// Source: ./interactive.go
//
// Generated by this command:
//
//	mockgen -source=./interactive.go -destination=../../mocks/interactive.mock.go -package=intrmocks -typed InteractiveService
//

// Package intrmocks is a generated GoMock package.
package intrmocks

import (
	context "context"
	reflect "reflect"

	domain "github.com/ecodeclub/webook/internal/interactive/internal/domain"
	gomock "go.uber.org/mock/gomock"
)

// MockService is a mock of Service interface.
type MockService struct {
	ctrl     *gomock.Controller
	recorder *MockServiceMockRecorder
}

// MockServiceMockRecorder is the mock recorder for MockService.
type MockServiceMockRecorder struct {
	mock *MockService
}

// NewMockService creates a new mock instance.
func NewMockService(ctrl *gomock.Controller) *MockService {
	mock := &MockService{ctrl: ctrl}
	mock.recorder = &MockServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockService) EXPECT() *MockServiceMockRecorder {
	return m.recorder
}

// CollectToggle mocks base method.
func (m *MockService) CollectToggle(ctx context.Context, biz string, bizId, uid int64) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CollectToggle", ctx, biz, bizId, uid)
	ret0, _ := ret[0].(error)
	return ret0
}

// CollectToggle indicates an expected call of CollectToggle.
func (mr *MockServiceMockRecorder) CollectToggle(ctx, biz, bizId, uid any) *MockServiceCollectToggleCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CollectToggle", reflect.TypeOf((*MockService)(nil).CollectToggle), ctx, biz, bizId, uid)
	return &MockServiceCollectToggleCall{Call: call}
}

// MockServiceCollectToggleCall wrap *gomock.Call
type MockServiceCollectToggleCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockServiceCollectToggleCall) Return(arg0 error) *MockServiceCollectToggleCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockServiceCollectToggleCall) Do(f func(context.Context, string, int64, int64) error) *MockServiceCollectToggleCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockServiceCollectToggleCall) DoAndReturn(f func(context.Context, string, int64, int64) error) *MockServiceCollectToggleCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// CollectionInfo mocks base method.
func (m *MockService) CollectionInfo(ctx context.Context, uid, id int64, offset, limit int) ([]domain.CollectionRecord, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CollectionInfo", ctx, uid, id, offset, limit)
	ret0, _ := ret[0].([]domain.CollectionRecord)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CollectionInfo indicates an expected call of CollectionInfo.
func (mr *MockServiceMockRecorder) CollectionInfo(ctx, uid, id, offset, limit any) *MockServiceCollectionInfoCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CollectionInfo", reflect.TypeOf((*MockService)(nil).CollectionInfo), ctx, uid, id, offset, limit)
	return &MockServiceCollectionInfoCall{Call: call}
}

// MockServiceCollectionInfoCall wrap *gomock.Call
type MockServiceCollectionInfoCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockServiceCollectionInfoCall) Return(arg0 []domain.CollectionRecord, arg1 error) *MockServiceCollectionInfoCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockServiceCollectionInfoCall) Do(f func(context.Context, int64, int64, int, int) ([]domain.CollectionRecord, error)) *MockServiceCollectionInfoCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockServiceCollectionInfoCall) DoAndReturn(f func(context.Context, int64, int64, int, int) ([]domain.CollectionRecord, error)) *MockServiceCollectionInfoCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// CollectionList mocks base method.
func (m *MockService) CollectionList(ctx context.Context, uid int64, offset, limit int) ([]domain.Collection, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CollectionList", ctx, uid, offset, limit)
	ret0, _ := ret[0].([]domain.Collection)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CollectionList indicates an expected call of CollectionList.
func (mr *MockServiceMockRecorder) CollectionList(ctx, uid, offset, limit any) *MockServiceCollectionListCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CollectionList", reflect.TypeOf((*MockService)(nil).CollectionList), ctx, uid, offset, limit)
	return &MockServiceCollectionListCall{Call: call}
}

// MockServiceCollectionListCall wrap *gomock.Call
type MockServiceCollectionListCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockServiceCollectionListCall) Return(arg0 []domain.Collection, arg1 error) *MockServiceCollectionListCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockServiceCollectionListCall) Do(f func(context.Context, int64, int, int) ([]domain.Collection, error)) *MockServiceCollectionListCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockServiceCollectionListCall) DoAndReturn(f func(context.Context, int64, int, int) ([]domain.Collection, error)) *MockServiceCollectionListCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// DeleteCollection mocks base method.
func (m *MockService) DeleteCollection(ctx context.Context, uid, id int64) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteCollection", ctx, uid, id)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteCollection indicates an expected call of DeleteCollection.
func (mr *MockServiceMockRecorder) DeleteCollection(ctx, uid, id any) *MockServiceDeleteCollectionCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteCollection", reflect.TypeOf((*MockService)(nil).DeleteCollection), ctx, uid, id)
	return &MockServiceDeleteCollectionCall{Call: call}
}

// MockServiceDeleteCollectionCall wrap *gomock.Call
type MockServiceDeleteCollectionCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockServiceDeleteCollectionCall) Return(arg0 error) *MockServiceDeleteCollectionCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockServiceDeleteCollectionCall) Do(f func(context.Context, int64, int64) error) *MockServiceDeleteCollectionCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockServiceDeleteCollectionCall) DoAndReturn(f func(context.Context, int64, int64) error) *MockServiceDeleteCollectionCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// Get mocks base method.
func (m *MockService) Get(ctx context.Context, biz string, id, uid int64) (domain.Interactive, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", ctx, biz, id, uid)
	ret0, _ := ret[0].(domain.Interactive)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockServiceMockRecorder) Get(ctx, biz, id, uid any) *MockServiceGetCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockService)(nil).Get), ctx, biz, id, uid)
	return &MockServiceGetCall{Call: call}
}

// MockServiceGetCall wrap *gomock.Call
type MockServiceGetCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockServiceGetCall) Return(arg0 domain.Interactive, arg1 error) *MockServiceGetCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockServiceGetCall) Do(f func(context.Context, string, int64, int64) (domain.Interactive, error)) *MockServiceGetCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockServiceGetCall) DoAndReturn(f func(context.Context, string, int64, int64) (domain.Interactive, error)) *MockServiceGetCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// GetByIds mocks base method.
func (m *MockService) GetByIds(ctx context.Context, biz string, uid int64, ids []int64) (map[int64]domain.Interactive, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetByIds", ctx, biz, uid, ids)
	ret0, _ := ret[0].(map[int64]domain.Interactive)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetByIds indicates an expected call of GetByIds.
func (mr *MockServiceMockRecorder) GetByIds(ctx, biz, uid, ids any) *MockServiceGetByIdsCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetByIds", reflect.TypeOf((*MockService)(nil).GetByIds), ctx, biz, uid, ids)
	return &MockServiceGetByIdsCall{Call: call}
}

// MockServiceGetByIdsCall wrap *gomock.Call
type MockServiceGetByIdsCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockServiceGetByIdsCall) Return(arg0 map[int64]domain.Interactive, arg1 error) *MockServiceGetByIdsCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockServiceGetByIdsCall) Do(f func(context.Context, string, int64, []int64) (map[int64]domain.Interactive, error)) *MockServiceGetByIdsCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockServiceGetByIdsCall) DoAndReturn(f func(context.Context, string, int64, []int64) (map[int64]domain.Interactive, error)) *MockServiceGetByIdsCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// IncrReadCnt mocks base method.
func (m *MockService) IncrReadCnt(ctx context.Context, biz string, bizId int64) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IncrReadCnt", ctx, biz, bizId)
	ret0, _ := ret[0].(error)
	return ret0
}

// IncrReadCnt indicates an expected call of IncrReadCnt.
func (mr *MockServiceMockRecorder) IncrReadCnt(ctx, biz, bizId any) *MockServiceIncrReadCntCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IncrReadCnt", reflect.TypeOf((*MockService)(nil).IncrReadCnt), ctx, biz, bizId)
	return &MockServiceIncrReadCntCall{Call: call}
}

// MockServiceIncrReadCntCall wrap *gomock.Call
type MockServiceIncrReadCntCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockServiceIncrReadCntCall) Return(arg0 error) *MockServiceIncrReadCntCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockServiceIncrReadCntCall) Do(f func(context.Context, string, int64) error) *MockServiceIncrReadCntCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockServiceIncrReadCntCall) DoAndReturn(f func(context.Context, string, int64) error) *MockServiceIncrReadCntCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// LikeToggle mocks base method.
func (m *MockService) LikeToggle(c context.Context, biz string, id, uid int64) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LikeToggle", c, biz, id, uid)
	ret0, _ := ret[0].(error)
	return ret0
}

// LikeToggle indicates an expected call of LikeToggle.
func (mr *MockServiceMockRecorder) LikeToggle(c, biz, id, uid any) *MockServiceLikeToggleCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LikeToggle", reflect.TypeOf((*MockService)(nil).LikeToggle), c, biz, id, uid)
	return &MockServiceLikeToggleCall{Call: call}
}

// MockServiceLikeToggleCall wrap *gomock.Call
type MockServiceLikeToggleCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c_2 *MockServiceLikeToggleCall) Return(arg0 error) *MockServiceLikeToggleCall {
	c_2.Call = c_2.Call.Return(arg0)
	return c_2
}

// Do rewrite *gomock.Call.Do
func (c_2 *MockServiceLikeToggleCall) Do(f func(context.Context, string, int64, int64) error) *MockServiceLikeToggleCall {
	c_2.Call = c_2.Call.Do(f)
	return c_2
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c_2 *MockServiceLikeToggleCall) DoAndReturn(f func(context.Context, string, int64, int64) error) *MockServiceLikeToggleCall {
	c_2.Call = c_2.Call.DoAndReturn(f)
	return c_2
}

// MoveToCollection mocks base method.
func (m *MockService) MoveToCollection(ctx context.Context, biz string, bizid, uid, collectionId int64) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MoveToCollection", ctx, biz, bizid, uid, collectionId)
	ret0, _ := ret[0].(error)
	return ret0
}

// MoveToCollection indicates an expected call of MoveToCollection.
func (mr *MockServiceMockRecorder) MoveToCollection(ctx, biz, bizid, uid, collectionId any) *MockServiceMoveToCollectionCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MoveToCollection", reflect.TypeOf((*MockService)(nil).MoveToCollection), ctx, biz, bizid, uid, collectionId)
	return &MockServiceMoveToCollectionCall{Call: call}
}

// MockServiceMoveToCollectionCall wrap *gomock.Call
type MockServiceMoveToCollectionCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockServiceMoveToCollectionCall) Return(arg0 error) *MockServiceMoveToCollectionCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockServiceMoveToCollectionCall) Do(f func(context.Context, string, int64, int64, int64) error) *MockServiceMoveToCollectionCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockServiceMoveToCollectionCall) DoAndReturn(f func(context.Context, string, int64, int64, int64) error) *MockServiceMoveToCollectionCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// SaveCollection mocks base method.
func (m *MockService) SaveCollection(ctx context.Context, collection domain.Collection) (int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SaveCollection", ctx, collection)
	ret0, _ := ret[0].(int64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SaveCollection indicates an expected call of SaveCollection.
func (mr *MockServiceMockRecorder) SaveCollection(ctx, collection any) *MockServiceSaveCollectionCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SaveCollection", reflect.TypeOf((*MockService)(nil).SaveCollection), ctx, collection)
	return &MockServiceSaveCollectionCall{Call: call}
}

// MockServiceSaveCollectionCall wrap *gomock.Call
type MockServiceSaveCollectionCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockServiceSaveCollectionCall) Return(arg0 int64, arg1 error) *MockServiceSaveCollectionCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockServiceSaveCollectionCall) Do(f func(context.Context, domain.Collection) (int64, error)) *MockServiceSaveCollectionCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockServiceSaveCollectionCall) DoAndReturn(f func(context.Context, domain.Collection) (int64, error)) *MockServiceSaveCollectionCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}
