package generator

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestJWTTokenGen_GenerateToken(t *testing.T) {
	j := &JWTTokenGen{
		key:    "key",
		issuer: "test",
		nowFunc: func() time.Time {
			return time.Unix(1688443200, 0)
		},
	}
	tests := []struct {
		name    string
		subject string
		expire  time.Duration
		want    string
		wantErr error
	}{
		{
			name:    "生成token",
			subject: "foo@example.com",
			expire:  15 * time.Minute,
			want:    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJ0ZXN0Iiwic3ViIjoiZm9vQGV4YW1wbGUuY29tIiwiZXhwIjoxNjg4NDQ0MTAwLCJpYXQiOjE2ODg0NDMyMDB9.c89aTGtb4uSsVHhUMyskJck2mbxGC_ELRCQv3Lt_dSs",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := j.GenerateToken(tt.subject, tt.expire)
			assert.Equal(t, tt.wantErr, err)
			if token != tt.want {
				t.Errorf("wrong token generated. want: %q; got: %q", tt.want, token)
			}
		})
	}
}
