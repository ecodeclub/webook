package snowflake

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_NewGenerate(t *testing.T) {
	testcases := []struct {
		name        string
		nodeId      uint
		apps        uint
		wantErrFunc require.ErrorAssertionFunc
	}{
		{
			name:   "nodeId超出限制",
			nodeId: 32,
			apps:   6,
			wantErrFunc: func(t require.TestingT, err error, _ ...interface{}) {
				require.ErrorIs(t, err, ErrExceedNode)
			},
		},
		{
			name:   "appId超出限制",
			nodeId: 3,
			apps:   33,
			wantErrFunc: func(t require.TestingT, err error, _ ...interface{}) {
				require.ErrorIs(t, err, ErrExceedApp)
			},
		},
		{
			name:        "生成正常",
			nodeId:      0,
			apps:        6,
			wantErrFunc: require.NoError,
		},
	}
	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewMeoyingIDGenerator(tt.nodeId, tt.apps)
			tt.wantErrFunc(t, err)
		})
	}

}

func Test_Generate(t *testing.T) {
	idmaker, err := NewMeoyingIDGenerator(1, 6)
	require.NoError(t, err)
	ids := make([]int64, 0)
	for i := 0; i < 6; i++ {
		for j := 0; j < 100000; j++ {
			id, err := idmaker.Generate(uint(i))
			require.NoError(t, err)
			ids = append(ids, id.Int64())
		}
	}
	// 校验生成的id是否重复
	idmap := make(map[int64]struct{}, len(ids))
	for i := 0; i < len(ids); i++ {
		_, ok := idmap[ids[i]]
		require.False(t, ok)
		idmap[ids[i]] = struct{}{}
	}

}

func Test_GenerateAppId(t *testing.T) {
	idmaker, err := NewMeoyingIDGenerator(1, 16)
	require.NoError(t, err)
	testcases := []struct {
		name    string
		appid   uint
		wantErr require.ErrorAssertionFunc
	}{
		{
			name:  "appId没找到",
			appid: 16,
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.ErrorIs(t, err, ErrUnknownApp)
			},
		},
		{
			name:    "appid 为1",
			appid:   1,
			wantErr: require.NoError,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			id, err := idmaker.Generate(tc.appid)
			tc.wantErr(t, err)
			if err != nil {
				return
			}
			app := id.AppID()
			assert.Equal(t, tc.appid, app)
		})
	}
}
