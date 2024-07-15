package getter

import (
	"context"
	"testing"

	"github.com/ecodeclub/webook/internal/ai/internal/service/handler/gpt/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Polling(t *testing.T) {
	testcases := []struct {
		name         string
		sdks         []sdk.GPTSdk
		wantMockSdks []sdk.GPTSdk
		wantErr      error
	}{
		{
			name: "sdk轮询拿出",
			sdks: []sdk.GPTSdk{
				&MockSdk{
					index: 0,
				},
				&MockSdk{
					index: 1,
				},
				&MockSdk{
					index: 2,
				},
			},
			wantMockSdks: []sdk.GPTSdk{
				&MockSdk{
					index: 0,
				},
				&MockSdk{
					index: 1,
				},
				&MockSdk{
					index: 2,
				},
			},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			getter := NewPollingGetter(tc.wantMockSdks)
			for i := 0; i < len(tc.wantMockSdks); i++ {
				gsdk, err := getter.GetSdk("xxx")
				require.NoError(t, err)
				index, _, err := gsdk.Invoke(context.Background(), []string{})
				require.NoError(t, err)
				assert.Equal(t, index, int64(i))
			}
		})
	}
}

type MockSdk struct {
	index int64
}

func (m *MockSdk) Invoke(ctx context.Context, input []string) (int64, string, error) {
	return m.index, "", nil
}
