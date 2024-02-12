package test

import "github.com/ecodeclub/ekit/net/httpx/httptestx"

func NewJSONResponseRecorder[T any]() *httptestx.JSONResponseRecorder[Result[T]] {
	return httptestx.NewJSONResponseRecorder[Result[T]]()
}
