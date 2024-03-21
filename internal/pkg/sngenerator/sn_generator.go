package sngenerator

import (
	"fmt"
	"time"

	"github.com/lithammer/shortuuid/v4"
)

// TimestampGenerateFunc 定义生成时间戳的函数类型
type TimestampGenerateFunc func(time.Time) int64

// ShortUUIDGenerateFunc 定义生成ShortUUID的函数类型
type ShortUUIDGenerateFunc func() string

// SequenceNumberGenerator 包含时间和UUID生成函数
type SequenceNumberGenerator struct {
	timestampGenFunc TimestampGenerateFunc
	shortUUIDGenFunc ShortUUIDGenerateFunc
}

// NewSequenceNumberGeneratorWith 创建一个SequenceNumberGenerator实例
func NewSequenceNumberGeneratorWith(timestampGen TimestampGenerateFunc, uuidGen ShortUUIDGenerateFunc) *SequenceNumberGenerator {
	return &SequenceNumberGenerator{
		timestampGenFunc: timestampGen,
		shortUUIDGenFunc: uuidGen,
	}
}

// NewSequenceNumberGenerator 创建一个SequenceNumberGenerator实例
func NewSequenceNumberGenerator() *SequenceNumberGenerator {
	return NewSequenceNumberGeneratorWith(func(t time.Time) int64 { return t.UnixMilli() }, func() string { return shortuuid.New() })
}

// Generate 使用ID生成序列号
func (s *SequenceNumberGenerator) Generate(id int64) (string, error) {
	timestamp := s.timestampGenFunc(time.Now())
	lastFour := fmt.Sprintf("%04d", id%10000)
	uuid := s.shortUUIDGenFunc()
	return fmt.Sprintf("%d%s%s", timestamp, lastFour, uuid), nil
}
