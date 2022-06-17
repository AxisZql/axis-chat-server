package etcd

import (
	"axisChat/utils/zlog"
	"context"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/coreos/etcd/mvcc/mvccpb"
	"go.etcd.io/etcd/clientv3"
)

//ServiceDiscovery 服务发现
type ServiceDiscovery struct {
	cli        *clientv3.Client       //etcd client
	serverList map[string][]*Instance //注册到etcd的所有服务的列表，建立对应path和旗下所有服务的映射，一个serverId对应多个服务
	indexMap   map[string]int         // 记录上一次轮询获取服务实例的索引
	serverKey  map[string]int         // 记录所有所有服务对应的所有key，一个key对应一个服务，记录对应服务在Instance slice中的索引
	lock       sync.RWMutex
}

// Instance 服务实例
type Instance struct {
	Cancel context.CancelFunc
	Ctx    context.Context
	Conn   *grpc.ClientConn
}

//NewServiceDiscovery  新建发现服务
func NewServiceDiscovery(endpoints []string) *ServiceDiscovery {
	//初始化etcd client v3
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,       // etcd cluster address
		DialTimeout: 5 * time.Second, //最长超时失效时间
	})

	if err != nil {
		log.Fatal(err)
	}

	return &ServiceDiscovery{
		cli:        cli,
		serverList: make(map[string][]*Instance),
		indexMap:   make(map[string]int),
		serverKey:  make(map[string]int),
	}
}

//WatchService 初始化服务列表和监视
func (s *ServiceDiscovery) WatchService(prefix string) error {
	//根据前缀获取现有的key，获取具有prefix前缀的path下的对应服务
	resp, err := s.cli.Get(context.Background(), prefix, clientv3.WithPrefix())
	if err != nil {
		return err
	}

	//遍历获取到的key和value
	for _, ev := range resp.Kvs {
		// ev.Value是对应注册到etcd服务的地址
		s.SetServiceList(string(ev.Key), string(ev.Value))
	}

	//监视前缀，修改变更的server
	go s.watcher(prefix)
	return nil
}

//watcher 监听key的前缀
func (s *ServiceDiscovery) watcher(prefix string) {
	rch := s.cli.Watch(context.Background(), prefix, clientv3.WithPrefix())
	log.Printf("watching prefix:%s now...", prefix)
	// TODO：监听具有对应前缀path下面注册服务列表的变更操作
	for wresp := range rch {
		for _, ev := range wresp.Events {
			switch ev.Type {
			case mvccpb.PUT: //修改或者新增
				s.SetServiceList(string(ev.Kv.Key), string(ev.Kv.Value))
			case mvccpb.DELETE: //删除
				s.DelServiceList(string(ev.Kv.Key))
			}
		}
	}
}

//SetServiceList 新增服务地址
func (s *ServiceDiscovery) SetServiceList(key, val string) {
	var err error
	s.lock.Lock()
	defer s.lock.Unlock()
	var idx = -1
	if _, ok := s.serverKey[key]; !ok {
		ins := &Instance{}
		ins.Conn, err = grpc.Dial(val, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			zlog.Error(err.Error())
			return
		}
		ins.Ctx, ins.Cancel = context.WithTimeout(context.Background(), time.Second*5)
		// 如果是新增服务
		strList := strings.Split(key, "&")
		serverId := strings.Replace(strList[1], "serverId=", "", -1)
		if _, ok2 := s.serverList[serverId]; !ok2 {
			// 如果对应serverId上还没有服务则初始化对应的key
			s.serverList[serverId] = []*Instance{
				ins,
			}
			// 初始化轮询获取实例下标
			s.indexMap[serverId] = 0
			idx = 0
		} else {
			s.serverList[serverId] = append(s.serverList[serverId], ins)
			idx = len(s.serverList[serverId])
		}
	}
	// 记录对应服务的key,即对应服务在Instance slice 中的索引
	if idx != -1 {
		s.serverKey[key] = idx
	}
	log.Println("put key :", key, "val:", val)
}

//DelServiceList 删除服务地址
func (s *ServiceDiscovery) DelServiceList(key string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	idx := s.serverKey[key]
	strList := strings.Split(key, "&")
	serverId := strings.Replace(strList[1], "serverId=", "", -1)
	if idx > len(s.serverList[serverId]) {
		zlog.Error("服务索引不应该大于所有可用服务实例数")
	}
	// 删除serverId下对应key的一个服务
	// todo 先关闭rpc客户端
	gs := s.serverList[serverId][idx]
	gs.Cancel()
	err := gs.Conn.Close()
	if err != nil {
		zlog.Error(err.Error())
	}

	tmp := s.serverList[serverId][idx+1:]
	s.serverList[serverId] = append(s.serverList[serverId][:idx], tmp...)
	log.Println("del key:", key)
}

// GetServiceByServerId 根据serverId获取服务实例
func (s *ServiceDiscovery) GetServiceByServerId(serverId string) (ins *Instance, err error) {
	//todo 对于logic layer 所有的logic层服务人为指定一个serverId，所有的logic 实例都使用这个serverId
	s.lock.RLock()
	defer s.lock.RUnlock()
	_, ok := s.serverList[serverId]
	if !ok {
		err = errors.New("the serverId does not exist!!!")
		return
	}
	// 开始轮询获取同一个serverId下的实例
	idx := s.indexMap[serverId] % len(s.serverList[serverId])
	ins = s.serverList[serverId][idx]
	// 设置下一次轮询的索引
	s.indexMap[serverId] = (s.indexMap[serverId] + 1) % len(s.serverList[serverId])
	return
}

//Close 关闭服务
func (s *ServiceDiscovery) Close() error {
	return s.cli.Close()
}
