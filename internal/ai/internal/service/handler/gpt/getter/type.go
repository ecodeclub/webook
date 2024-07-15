package getter

import "github.com/ecodeclub/webook/internal/ai/internal/service/handler/gpt/sdk"

type AiSdkGetter interface {
	GetSdk(biz string) (sdk.GPTSdk, error)
}
