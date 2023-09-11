// Code generated by MockGen. DO NOT EDIT.
// Source: internal/repository/email.go

// Package repomocks is a generated GoMock package.
package repomocks

import (
	context "context"
	reflect "reflect"

	domain "github.com/ecodeclub/webook/internal/domain"
	dao "github.com/ecodeclub/webook/internal/repository/dao"
	gomock "go.uber.org/mock/gomock"
)

// MockEamilRepository is a mock of EamilRepository interface.
type MockEamilRepository struct {
	ctrl     *gomock.Controller
	recorder *MockEamilRepositoryMockRecorder
}

// MockEamilRepositoryMockRecorder is the mock recorder for MockEamilRepository.
type MockEamilRepositoryMockRecorder struct {
	mock *MockEamilRepository
}

// NewMockEamilRepository creates a new mock instance.
func NewMockEamilRepository(ctrl *gomock.Controller) *MockEamilRepository {
	mock := &MockEamilRepository{ctrl: ctrl}
	mock.recorder = &MockEamilRepositoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockEamilRepository) EXPECT() *MockEamilRepositoryMockRecorder {
	return m.recorder
}

// FindByEmail mocks base method.
func (m *MockEamilRepository) FindByEmail(ctx context.Context, email string) (dao.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindByEmail", ctx, email)
	ret0, _ := ret[0].(dao.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindByEmail indicates an expected call of FindByEmail.
func (mr *MockEamilRepositoryMockRecorder) FindByEmail(ctx, email interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindByEmail", reflect.TypeOf((*MockEamilRepository)(nil).FindByEmail), ctx, email)
}

// Update mocks base method.
func (m *MockEamilRepository) Update(ctx context.Context, u domain.User) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", ctx, u)
	ret0, _ := ret[0].(error)
	return ret0
}

// Update indicates an expected call of Update.
func (mr *MockEamilRepositoryMockRecorder) Update(ctx, u interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockEamilRepository)(nil).Update), ctx, u)
}
