// Code generated by MockGen. DO NOT EDIT.
// Source: internal/repository/dao/user.go

// Package sqlmock is a generated GoMock package.
package sqlmock

import (
	context "context"
	reflect "reflect"

	dao "github.com/ecodeclub/webook/internal/repository/dao"
	gomock "go.uber.org/mock/gomock"
)

// MockUserDAO is a mock of UserDAO interface.
type MockUserDAO struct {
	ctrl     *gomock.Controller
	recorder *MockUserDAOMockRecorder
}

// MockUserDAOMockRecorder is the mock recorder for MockUserDAO.
type MockUserDAOMockRecorder struct {
	mock *MockUserDAO
}

// NewMockUserDAO creates a new mock instance.
func NewMockUserDAO(ctrl *gomock.Controller) *MockUserDAO {
	mock := &MockUserDAO{ctrl: ctrl}
	mock.recorder = &MockUserDAOMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockUserDAO) EXPECT() *MockUserDAOMockRecorder {
	return m.recorder
}

// FindByEmail mocks base method.
func (m *MockUserDAO) FindByEmail(ctx context.Context, email string) (dao.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindByEmail", ctx, email)
	ret0, _ := ret[0].(dao.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindByEmail indicates an expected call of FindByEmail.
func (mr *MockUserDAOMockRecorder) FindByEmail(ctx, email interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindByEmail", reflect.TypeOf((*MockUserDAO)(nil).FindByEmail), ctx, email)
}

// FindById mocks base method.
func (m *MockUserDAO) FindById(ctx context.Context, id int64) (dao.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindById", ctx, id)
	ret0, _ := ret[0].(dao.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindById indicates an expected call of FindById.
func (mr *MockUserDAOMockRecorder) FindById(ctx, id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindById", reflect.TypeOf((*MockUserDAO)(nil).FindById), ctx, id)
}

// Insert mocks base method.
func (m *MockUserDAO) Insert(ctx context.Context, u dao.User) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Insert", ctx, u)
	ret0, _ := ret[0].(error)
	return ret0
}

// Insert indicates an expected call of Insert.
func (mr *MockUserDAOMockRecorder) Insert(ctx, u interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Insert", reflect.TypeOf((*MockUserDAO)(nil).Insert), ctx, u)
}

// UpdateEmailVerified mocks base method.
func (m *MockUserDAO) UpdateEmailVerified(ctx context.Context, email string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateEmailVerified", ctx, email)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateEmailVerified indicates an expected call of UpdateEmailVerified.
func (mr *MockUserDAOMockRecorder) UpdateEmailVerified(ctx, email interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateEmailVerified", reflect.TypeOf((*MockUserDAO)(nil).UpdateEmailVerified), ctx, email)
}

// UpdateUserProfile mocks base method.
func (m *MockUserDAO) UpdateUserProfile(ctx context.Context, u dao.User) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateUserProfile", ctx, u)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateUserProfile indicates an expected call of UpdateUserProfile.
func (mr *MockUserDAOMockRecorder) UpdateUserProfile(ctx, u interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateUserProfile", reflect.TypeOf((*MockUserDAO)(nil).UpdateUserProfile), ctx, u)
}
