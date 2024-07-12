// Code generated by MockGen. DO NOT EDIT.
// Source: ./gpt.go
//
// Generated by this command:
//
//	mockgen -source=./gpt.go -destination=../../mocks/gpt.mock.go -package=aimocks -typed=true GPTService
//
// Package aimocks is a generated GoMock package.
package aimocks

import (
	context "context"
	reflect "reflect"

	service "github.com/ecodeclub/webook/internal/ai/internal/service"
	gomock "go.uber.org/mock/gomock"
)

// MockGPTService is a mock of GPTService interface.
type MockGPTService struct {
	ctrl     *gomock.Controller
	recorder *MockGPTServiceMockRecorder
}

// MockGPTServiceMockRecorder is the mock recorder for MockGPTService.
type MockGPTServiceMockRecorder struct {
	mock *MockGPTService
}

// NewMockGPTService creates a new mock instance.
func NewMockGPTService(ctrl *gomock.Controller) *MockGPTService {
	mock := &MockGPTService{ctrl: ctrl}
	mock.recorder = &MockGPTServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockGPTService) EXPECT() *MockGPTServiceMockRecorder {
	return m.recorder
}

// Invoke mocks base method.
func (m *MockGPTService) Invoke(ctx context.Context, req service.GPTRequest) (service.GPTResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Invoke", ctx, req)
	ret0, _ := ret[0].(service.GPTResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Invoke indicates an expected call of Invoke.
func (mr *MockGPTServiceMockRecorder) Invoke(ctx, req any) *GPTServiceInvokeCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Invoke", reflect.TypeOf((*MockGPTService)(nil).Invoke), ctx, req)
	return &GPTServiceInvokeCall{Call: call}
}

// GPTServiceInvokeCall wrap *gomock.Call
type GPTServiceInvokeCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *GPTServiceInvokeCall) Return(arg0 service.GPTResponse, arg1 error) *GPTServiceInvokeCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *GPTServiceInvokeCall) Do(f func(context.Context, service.GPTRequest) (service.GPTResponse, error)) *GPTServiceInvokeCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *GPTServiceInvokeCall) DoAndReturn(f func(context.Context, service.GPTRequest) (service.GPTResponse, error)) *GPTServiceInvokeCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}
