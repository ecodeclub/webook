package snowflake

import (
	"github.com/bwmarrin/snowflake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_NewGenerate(t *testing.T) {
	t.Run("nodeId超出限制", func(t *testing.T) {
		_, err := NewCustomSnowFlake(32, 6)
		require.ErrorIs(t, err, ErrExceedNode)
	})
	t.Run("app数量超出限制", func(t *testing.T) {
		_, err := NewCustomSnowFlake(3, 33)
		require.ErrorIs(t, err, ErrExceedApp)
	})
	t.Run("正常生成", func(t *testing.T) {
		idMaker, err := NewCustomSnowFlake(0, 4)
		require.NoError(t, err)
		appids := make([]uint, 0, 4)
		idMaker.nodes.Range(func(key uint, value *snowflake.Node) bool {
			appids = append(appids, key)
			return true
		})
		assert.ElementsMatch(t, []uint{
			0, 1, 2, 3,
		}, appids)
	})

}

func Test_Generate(t *testing.T) {
	idmaker, err := NewCustomSnowFlake(1, 6)
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
	idmaker, err := NewCustomSnowFlake(1, 16)
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
