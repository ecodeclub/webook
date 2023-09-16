package retry

import "time"

type Strategy interface {
	// Next 返回下一次重试的间隔，如果不需要继续重试，那么第二参数返回 false
	Next() (time.Duration, bool)
}
