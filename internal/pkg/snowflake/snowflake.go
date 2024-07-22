package snowflake

import (
	"errors"
	"fmt"

	"github.com/bwmarrin/snowflake"
	"github.com/ecodeclub/ekit/syncx"
)

type ID int64

// AppID 返回生成时输入的appid
func (f ID) AppID() uint {
	node := snowflake.ID(f).Node()
	return uint(node >> 5)
}

func (f ID) Int64() int64 {
	return int64(f)
}

type AppIDGenerator interface {
	// Generate 功能生成雪花id（ID）。返回雪花id（ID）的每一位的组成如下。返回值ID可以通过AppID()返回生成时输入的appid。
	// +---------------------------------------------------------------------------------------+
	// | 1 Bit Unused | 41 Bit Timestamp |  5 Bit APPID | 5 Bit NodeID  |   12 Bit Sequence ID |
	// +---------------------------------------------------------------------------------------+
	Generate(appid uint) (ID, error)
}

type MeoyingIDGenerator struct {
	// 键为appid
	nodes *syncx.Map[uint, *snowflake.Node]
}

const (
	maxNodeNum uint = 31
	maxAppNum  uint = 31
)

var (
	ErrExceedNode = errors.New("node编号超出限制")
	ErrExceedApp  = errors.New("app编号超出限制")
	ErrUnknownApp = errors.New("未知的app")
)

// NewMeoyingIDGenerator nodeId表示第几个节点，apps表示有几个应用 从0开始排序 0-webook 1-ielts 最多到31
func NewMeoyingIDGenerator(nodeId uint, apps uint) (*MeoyingIDGenerator, error) {
	nodeMap := &syncx.Map[uint, *snowflake.Node]{}
	if nodeId > maxNodeNum {
		return nil, fmt.Errorf("%w", ErrExceedNode)
	}
	if apps > maxAppNum+1 {
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
	return &MeoyingIDGenerator{
		nodes: nodeMap,
	}, nil

}

func (c *MeoyingIDGenerator) Generate(appid uint) (ID, error) {
	n, ok := c.nodes.Load(appid)
	if !ok {
		return 0, fmt.Errorf("%w", ErrUnknownApp)
	}
	id := n.Generate()
	return ID(id), nil
}
