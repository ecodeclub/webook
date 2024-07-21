package snowflake

import (
	"errors"
	"fmt"
	"github.com/bwmarrin/snowflake"
	"github.com/ecodeclub/ekit/syncx"
)

type SnowFlake interface {
	Generate(appid uint) (ID, error)
}

type CustomSnowFlake struct {
	// 键为appid
	nodes syncx.Map[uint, *snowflake.Node]
}

const (
	maxNode uint = 31
	maxApp  uint = 31
)

var (
	ErrExceedNode = errors.New("node超出限制")
	ErrExceedApp  = errors.New("app超出限制")
	ErrUnknownApp = errors.New("未知的app")
)

// +---------------------------------------------------------------------------------------+
// | 1 Bit Unused | 41 Bit Timestamp |  5 Bit APPID | 5 Bit NodeID  |   12 Bit Sequence ID |
// +---------------------------------------------------------------------------------------+

// node表示第几个节点，appid表示有几个应用 从0开始排序  0-ietls 最多到31
func NewCustomSnowFlake(nodeId uint, apps uint) (*CustomSnowFlake, error) {
	nodeMap := syncx.Map[uint, *snowflake.Node]{}
	if nodeId > maxNode {
		return nil, fmt.Errorf("%w", ErrExceedNode)
	}
	if apps > maxApp+1 {
		return nil, fmt.Errorf("%w", ErrExceedApp)
	}
	for i := 0; i < int(apps); i++ {
		nid := (i << 5) | int(nodeId)
		n, err := snowflake.NewNode(int64(nid))
		if err != nil {
			return nil, err
		}
		nodeMap.Store(uint(i), n)
	}
	return &CustomSnowFlake{
		nodes: nodeMap,
	}, nil

}

type ID int64

func (c *CustomSnowFlake) Generate(appid uint) (ID, error) {
	n, ok := c.nodes.Load(appid)
	if !ok {
		return 0, fmt.Errorf("%w", ErrUnknownApp)
	}
	id := n.Generate()
	return ID(id), nil
}

func (f ID) AppID() uint {
	node := snowflake.ID(f).Node()
	return uint(node >> 5)
}

func (f ID) Int64() int64 {
	return int64(f)
}
