//go:build manual

package goemail

import (
	"context"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/gomail.v2"
)

func TestService_Send(t *testing.T) {
	host := os.Getenv("HOST")
	portStr := os.Getenv("PORT")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatal(err)
	}
	username := os.Getenv("UNAME")
	password := os.Getenv("PASSWORD")
	d, err := gomail.NewDialer(host, port, username, password).Dial()
	if err != nil {
		t.Fatal(err)
	}
	s := NewService(username, d)
	tests := []struct {
		name    string
		ctx     context.Context
		from    string
		subject string
		body    string
		to      string
		wantErr error
	}{
		{
			name:    "发送",
			ctx:     context.Background(),
			from:    os.Getenv("FROM"),
			subject: "主题",
			body:    "内容",
			to:      os.Getenv("TO"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.Send(tt.ctx, tt.from, tt.subject, tt.body, tt.to, false)
			assert.Equal(t, tt.wantErr, err)
		})
	}
}
