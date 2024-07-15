package sdk

import "context"

//go:generate mockgen -source=./type.go -destination=../mocks/gpt.mock.go -package=aimocks -typed=true GPTSdk

// 各个ai sdk统一的抽象
type GPTSdk interface {
	// 返回值 第一个是token数，第二个为返回内容
	Invoke(ctx context.Context, input []string) (int64, string, error)
}
