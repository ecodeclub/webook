package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAddAppId(t *testing.T) {
	testCases := []struct {
		name      string
		wantCode  int
		before    func(t *testing.T, ctx *gin.Context)
		afterFunc func(t *testing.T, ctx *gin.Context)
	}{
		{
			name:     "appid 为 1",
			wantCode: 200,
			before: func(t *testing.T, ctx *gin.Context) {
				header := make(http.Header)
				header.Set(appCtxKey, "1")
				ctx.Request = httptest.NewRequest(http.MethodPost, "/users/profile", nil)
				ctx.Request.Header = header
			},
			afterFunc: func(t *testing.T, ctx *gin.Context) {
				c := ctx.Request.Context()
				v := c.Value(appCtxKey)
				res, ok := v.(uint)
				require.True(t, ok)
				assert.Equal(t, uint(1), res)
			},
		},
		{
			name:     "appid没设置",
			wantCode: 200,
			before: func(t *testing.T, ctx *gin.Context) {
				header := make(http.Header)
				ctx.Request = httptest.NewRequest(http.MethodPost, "/users/profile", nil)
				ctx.Request.Header = header
			},
			afterFunc: func(t *testing.T, ctx *gin.Context) {
				c := ctx.Request.Context()
				v := c.Value(appCtxKey)
				require.Nil(t, v)
			},
		},
		{
			name:     "appid 设置为不是数字",
			wantCode: 400,
			before: func(t *testing.T, ctx *gin.Context) {
				header := make(http.Header)
				header.Set(appCtxKey, "dasdsa")
				ctx.Request = httptest.NewRequest(http.MethodPost, "/users/profile", nil)
				ctx.Request.Header = header
			},
			afterFunc: func(t *testing.T, ctx *gin.Context) {
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			tc.before(t, c)
			builder := NewAddAppIdBuilder()
			hdl := builder.Build()
			hdl(c)
			assert.Equal(t, tc.wantCode, c.Writer.Status())
			tc.afterFunc(t, c)
		})
	}
}
