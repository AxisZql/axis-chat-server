package utils

import (
	"fmt"
	"github.com/pkg/errors"
	"hash/crc32"
	"sort"
	"sync"
)

/**
*Author:AxisZql
*Date:2022-6-1
*DESC:使用一致性哈希来保证使用多个redis服务时确保服务的高可靠性
 */

// Hash define to generate hash function
type Hash func(data []byte) uint32

type Uint32Slice []uint32

func (s Uint32Slice) Len() int {
	return len(s)
}

func (s Uint32Slice) Less(i, j int) bool {
	return s[i] < s[j]
}

func (s Uint32Slice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// HashBalance 一致性哈希数据结构
type HashBalance struct {
	mux      sync.RWMutex      //引入锁
	hash     Hash              // 计算哈希的哈希函数，可以自定义
	replicas int               // 复制因子（也就是一台主机对应的虚拟节点的个数）
	keys     Uint32Slice       // 已经排序的哈希切片，（由于一致性哈希的空间为2^32-1,故用uint32来记录一个哈希值)
	hashMap  map[uint32]string // 节点哈希和key的map，键值是hash值，值是节点key

	//其他配置，维护一致性哈希的观察主体
	conf LoadBalanceConf
}
type LoadBalanceConf struct{}

// NewHashBalance 构造一致性哈希对象
func NewHashBalance(replies int, fn Hash) *HashBalance {
	m := &HashBalance{
		replicas: replies,
		hash:     fn,
		hashMap:  make(map[uint32]string),
	}
	// 如果fn==nil，则使用默认的crc32.ChecksumIEEE 哈希函数
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

func (c *HashBalance) SetConf(conf LoadBalanceConf) {
	c.conf = conf
}

func (c *HashBalance) IsEmpty() bool {
	return len(c.keys) == 0
}

// Add 添加缓存节点，参数为节点的key，比如使用ip@port
func (c *HashBalance) Add(params ...string) error {
	if len(params) == 0 {
		return errors.New("param len 1 at least")
	}
	// 加写锁
	c.mux.Lock()
	defer c.mux.Unlock()
	for _, val := range params {
		for i := 0; i < c.replicas; i++ {
			// 每个虚拟节点的key规则为ip:port#[0-9]
			hash := c.hash([]byte(fmt.Sprintf("%s#%d", val, i)))
			c.keys = append(c.keys, hash)
			c.hashMap[hash] = val // 虚拟节点的哈希值都通过hashMap映射到真实节点上
		}
	}
	// 对所有虚拟节点的哈希值进行排序，方便之后进行二分查找
	sort.Sort(c.keys)
	return nil
}

// Get 方法根据客户端请求对应的key来计算哈希，通过hash获取离自己哈希最近的虚拟节点
func (c *HashBalance) Get(key string) (string, error) {
	if c.IsEmpty() {
		return "", errors.New("node is empty")
	}
	hash := c.hash([]byte(key))
	// 通过二分查找获取最优的节点，第一个“服务器hash”值大于“数据hash”的就是最优“服务器节点”
	// 如果目标数组没有对应的数据sort.Search返回的是该数组的长度
	idx := sort.Search(len(c.keys), func(i int) bool { return c.keys[i] >= hash })
	// 如果查找结果大于服务器哈希数组的最大索引，表示此时该对象的哈希值位于最后一个节点之后，
	//由于是顺时针遇到的第一个节点，所以该对应应该放入第一个服务器节点中
	if idx == len(c.keys) {
		idx = 0
	}
	// 防止读数据是hashMap的值改变，故加读锁
	c.mux.RLock()
	defer c.mux.RUnlock()
	return c.hashMap[c.keys[idx]], nil
}
