// Code generated by MockGen. DO NOT EDIT.
// Source: ./notify.go
//
// Generated by this command:
//
//	mockgen -source=./notify.go -package=wechatmocks -destination=./mocks/notify.mock.go -typed NotifyHandler
//

// Package wechatmocks is a generated GoMock package.
package wechatmocks

import (
	context "context"
	http "net/http"
	reflect "reflect"

	notify "github.com/wechatpay-apiv3/wechatpay-go/core/notify"
	gomock "go.uber.org/mock/gomock"
)

// MockNotifyHandler is a mock of NotifyHandler interface.
type MockNotifyHandler struct {
	ctrl     *gomock.Controller
	recorder *MockNotifyHandlerMockRecorder
	isgomock struct{}
}

// MockNotifyHandlerMockRecorder is the mock recorder for MockNotifyHandler.
type MockNotifyHandlerMockRecorder struct {
	mock *MockNotifyHandler
}

// NewMockNotifyHandler creates a new mock instance.
func NewMockNotifyHandler(ctrl *gomock.Controller) *MockNotifyHandler {
	mock := &MockNotifyHandler{ctrl: ctrl}
	mock.recorder = &MockNotifyHandlerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockNotifyHandler) EXPECT() *MockNotifyHandlerMockRecorder {
	return m.recorder
}

// ParseNotifyRequest mocks base method.
func (m *MockNotifyHandler) ParseNotifyRequest(ctx context.Context, request *http.Request, content any) (*notify.Request, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ParseNotifyRequest", ctx, request, content)
	ret0, _ := ret[0].(*notify.Request)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ParseNotifyRequest indicates an expected call of ParseNotifyRequest.
func (mr *MockNotifyHandlerMockRecorder) ParseNotifyRequest(ctx, request, content any) *MockNotifyHandlerParseNotifyRequestCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ParseNotifyRequest", reflect.TypeOf((*MockNotifyHandler)(nil).ParseNotifyRequest), ctx, request, content)
	return &MockNotifyHandlerParseNotifyRequestCall{Call: call}
}

// MockNotifyHandlerParseNotifyRequestCall wrap *gomock.Call
type MockNotifyHandlerParseNotifyRequestCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockNotifyHandlerParseNotifyRequestCall) Return(arg0 *notify.Request, arg1 error) *MockNotifyHandlerParseNotifyRequestCall {
	c.Call = c.Call.Return(arg0, arg1)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockNotifyHandlerParseNotifyRequestCall) Do(f func(context.Context, *http.Request, any) (*notify.Request, error)) *MockNotifyHandlerParseNotifyRequestCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockNotifyHandlerParseNotifyRequestCall) DoAndReturn(f func(context.Context, *http.Request, any) (*notify.Request, error)) *MockNotifyHandlerParseNotifyRequestCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}
