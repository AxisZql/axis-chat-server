package connect

import (
	"axisChat/utils/zlog"
	"github.com/segmentio/kafka-go"
	"sync"
)

type GroupNode struct {
	groupId     int64
	OnlineCount int  // 记录当前群聊的在线人数，当人数为0时，会从bucket中删除该节点
	drop        bool // 删除节点之前必须保证drop==true,以防止新建的节点被误删
	mutex       sync.RWMutex
	socketHead  *Channel // 记录当前群聊所有在线用户连接的双向链表
}

func NewGroupNode(groupId int64) *GroupNode {
	return &GroupNode{
		groupId:     groupId,
		OnlineCount: 0,
		drop:        false,
		socketHead:  nil,
	}
}

func (g *GroupNode) Put(ch *Channel) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	if g.drop {
		zlog.Debug("before put --the group node is drop")
	}
	if g.socketHead != nil {
		// 前插
		g.socketHead.Prev = ch
	}
	ch.Next = g.socketHead
	ch.Prev = nil
	g.socketHead = ch
	g.OnlineCount++
	g.drop = false
	return
}

// DeleteChannel 删除双向链表中的对应的Channel节点，该函数可以在用户下线和用户 TODO：退群时调用
func (g *GroupNode) DeleteChannel(ch *Channel) bool {
	g.mutex.RLock()
	if ch.Next != nil {
		// 如果不是尾
		ch.Next.Prev = ch.Prev
	}
	if ch.Prev != nil {
		ch.Prev.Next = ch.Next
	} else {
		// 如果是头
		g.socketHead = ch.Next
	}
	g.OnlineCount--
	g.drop = false
	if g.OnlineCount <= 0 {
		g.drop = true
	}
	g.mutex.RUnlock()
	return g.drop
}

func (g *GroupNode) PushGroupMsg(msg kafka.Message) {
	g.mutex.RLock()
	for ch := g.socketHead; ch != nil; ch = ch.Next {
		//to send msg
		ch.Push(msg)
	}
	g.mutex.RUnlock()
	return
}

func (g *GroupNode) PushGroupStatusMsg(msg []byte) {
	g.mutex.RLock()
	for ch := g.socketHead; ch != nil; ch = ch.Next {
		//to send msg
		ch.PushStatus(msg)
	}
	g.mutex.RUnlock()
	return

}
