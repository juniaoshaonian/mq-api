package memory

import (
	"sync"

	"github.com/ecodeclub/ekit/list"
	"github.com/ecodeclub/mq-api"
)

// Partition 表示分区 是并发安全的
const (
	defaultPartitionCap = 64
)

type Partition struct {
	locker sync.RWMutex
	data   *list.ArrayList[*mq.Message]
}

func NewPartition() *Partition {
	return &Partition{
		data: list.NewArrayList[*mq.Message](defaultPartitionCap),
	}
}

func (p *Partition) sendMsg(msg *mq.Message) {
	p.locker.Lock()
	defer p.locker.Unlock()
	msg.Offset = int64(p.data.Len())
	_ = p.data.Append(msg)
}

func (p *Partition) consumerMsg(cursor, limit int) []*mq.Message {
	p.locker.RLock()
	defer p.locker.RUnlock()
	wantLen := cursor + limit + 1
	length := min(wantLen, p.data.Len())
	res := p.data.AsSlice()[cursor:length]
	return res
}
