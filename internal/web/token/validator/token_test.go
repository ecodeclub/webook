package validator

import (
	"testing"
	"time"
)

func TestJWTTokenVerifier_Verify(t *testing.T) {
	j := &JWTTokenVerifier{
		Key: "key",
	}
	tests := []struct {
		name    string
		token   string
		now     time.Time
		want    string
		wantErr bool
	}{
		{
			name:  "有效token",
			token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJ0ZXN0Iiwic3ViIjoiZm9vQGV4YW1wbGUuY29tIiwiZXhwIjoxNjg4NDQ0MTAwLCJpYXQiOjE2ODg0NDMyMDB9.c89aTGtb4uSsVHhUMyskJck2mbxGC_ELRCQv3Lt_dSs",
			now:   time.Unix(1688443300, 0),
			want:  "foo@example.com",
		},
		{
			name:    "token已过期",
			token:   "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJ0ZXN0Iiwic3ViIjoiZm9vQGV4YW1wbGUuY29tIiwiZXhwIjoxNjg4NDQ0MTAwLCJpYXQiOjE2ODg0NDMyMDB9.c89aTGtb4uSsVHhUMyskJck2mbxGC_ELRCQv3Lt_dSs",
			now:     time.Unix(1688444200, 0),
			wantErr: true,
		},
		{
			name:    "错误token",
			now:     time.Unix(1688443600, 0),
			wantErr: true,
		},
		{
			name:    "签名错误",
			token:   "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJ0ZXN0Iiwic3ViIjoiZm9vQGV4YW1wbGUuY29tIiwiZXhwIjoxNjg4NDQ0MTAwLCJpYXQiOjE2ODg0NDMyMDB9.503uOlg8YtEhJsN3qbDeZevkMydk5_WQkKYRPMJ7X78",
			now:     time.Unix(1688443600, 0),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j.nowFunc = func() time.Time {
				return tt.now
			}
			subject, err := j.Verify(tt.token)
			if !tt.wantErr && err != nil {
				t.Errorf("verification failed: %v", err)
			}

			if tt.wantErr && err == nil {
				t.Errorf("want error; got no error")
			}

			if subject != tt.want {
				t.Errorf("wrong account id. want: %q, got: %q", tt.want, subject)
			}
		})
	}
}
