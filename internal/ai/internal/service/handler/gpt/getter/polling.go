package getter

import (
	"sync/atomic"

	"github.com/ecodeclub/webook/internal/ai/internal/service/handler/gpt/sdk"
)

// 轮询
type PollingGetter struct {
	Count int64
	Sdks  []sdk.GPTSdk
}

func NewPollingGetter(sdks []sdk.GPTSdk) *PollingGetter {
	return &PollingGetter{
		Sdks: sdks,
	}
}

func (p *PollingGetter) GetSdk(biz string) (sdk.GPTSdk, error) {
	res := p.Sdks[int(p.Count)%len(p.Sdks)]
	atomic.AddInt64(&p.Count, 1)
	return res, nil
}
