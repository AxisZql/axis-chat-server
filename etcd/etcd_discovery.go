package etcd

import (
	"axisChat/utils/zlog"
	"context"
	"fmt"
	"github.com/pkg/errors"
	"go.etcd.io/etcd/api/v3/mvccpb"
	"go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"strings"
	"sync"
	"time"
)

//ServiceDiscovery 服务发现
type ServiceDiscovery struct {
	cli        *clientv3.Client       //etcd client
	serverList map[string][]*Instance //注册到etcd的所有服务的列表，建立对应path和旗下所有服务的映射，一个serverId对应多个服务
	indexMap   map[string]int         // 记录上一次轮询获取服务实例的索引
	serverKey  map[string][]string    // 记录所有所有服务对应的所有key，一个key对应一个服务，记录对应服务在Instance slice中的索引,通过顺序维护目标key在Instance slice中的索引
	lock       sync.RWMutex
}

// Instance 服务实例
type Instance struct {
	Conn *grpc.ClientConn
}

//NewServiceDiscovery  新建发现服务
func NewServiceDiscovery(endpoints []string) *ServiceDiscovery {
	//初始化etcd client v3
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,        // etcd cluster address
		DialTimeout: 10 * time.Second, //最长超时失效时间
	})

	if err != nil {
		panic(err)
	}

	return &ServiceDiscovery{
		cli:        cli,
		serverList: make(map[string][]*Instance),
		indexMap:   make(map[string]int),
		serverKey:  make(map[string][]string),
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
	zlog.Info(fmt.Sprintf("watching prefix:%s now...", prefix))
	// TODO：监听具有对应前缀path下面注册服务列表的变更操作
	for wresp := range rch {
		for _, ev := range wresp.Events {
			switch ev.Type {
			case mvccpb.PUT: //修改或者新增
				zlog.Warn("新增对应实例")
				s.SetServiceList(string(ev.Kv.Key), string(ev.Kv.Value))
			case mvccpb.DELETE: //删除
				zlog.Warn("删除对应实例")
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
	strList := strings.Split(key, "&")
	serverId := strings.Replace(strList[1], "serverId=", "", -1)

	getElementIndexInSlice := func(key string, slice []string) int {
		for i, val := range slice {
			if val == key {
				return i
			}
		}
		return -1
	}

	if idx := getElementIndexInSlice(key, s.serverKey[serverId]); idx == -1 {
		ins := &Instance{}
		ins.Conn, err = grpc.Dial(val, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			zlog.Error(err.Error())
			return
		}
		// 如果是新增服务
		if _, ok2 := s.serverList[serverId]; !ok2 {
			// 如果对应serverId上还没有服务则初始化对应的key
			s.serverList[serverId] = []*Instance{
				ins,
			}
			// 建立key和Instance之间的映射
			s.serverKey[serverId] = []string{key}
			// 初始化轮询获取实例下标
			s.indexMap[serverId] = 0
		} else {
			s.serverList[serverId] = append(s.serverList[serverId], ins)
			s.serverKey[serverId] = append(s.serverKey[serverId], key)
			//idx = len(s.serverList[serverId])
		}
	}
	zlog.Info(fmt.Sprintf("put key :%s  val:%s", key, val))
}

//DelServiceList 删除服务地址
func (s *ServiceDiscovery) DelServiceList(key string) {
	s.lock.Lock()
	strList := strings.Split(key, "&")
	serverId := strings.Replace(strList[1], "serverId=", "", -1)

	getElementIndexInSlice := func(key string, slice []string) int {
		for i, val := range slice {
			if val == key {
				return i
			}
		}
		return -1
	}

	idx := getElementIndexInSlice(key, s.serverKey[serverId])
	if idx >= len(s.serverList[serverId]) {
		zlog.Error(fmt.Sprintf("服务索引不应该大于所有可用服务实例数「 idx=%d  len(serverList[serverId])=%d」", idx, len(s.serverList[serverId])))
	}
	// 删除serverId下对应key的一个服务
	// todo 先关闭rpc客户端
	gs := s.serverList[serverId][idx]

	tmp := s.serverList[serverId][idx+1:]
	s.serverList[serverId] = append(s.serverList[serverId][:idx], tmp...)

	//更新s.serverKey中的索引映射
	tmp2 := s.serverKey[serverId][idx+1:]
	s.serverKey[serverId] = append(s.serverKey[serverId][:idx], tmp2...)
	s.lock.Unlock()
	zlog.Info(fmt.Sprintf("del key:%s", key))

	err := gs.Conn.Close()
	if err != nil {
		zlog.Error(err.Error())
	}
}

// GetServiceByServerId 根据serverId获取服务实例
func (s *ServiceDiscovery) GetServiceByServerId(serverId string) (ins *Instance, err error) {
	//todo 对于logic layer 所有的logic层服务人为指定一个serverId，所有的logic 实例都使用这个serverId
	s.lock.RLock()
	_, ok := s.serverList[serverId]
	if !ok {
		err = errors.New("the serverId does not exist!!!")
		s.lock.RUnlock()
		return
	}
	if len(s.serverList[serverId]) == 0 {
		zlog.Error(fmt.Sprintf("no logic layer service instance are available!!!! 「len(s.serverList[serverId])=%d   len(s.serverKey[serverId])=%d」", len(s.serverList[serverId]), len(s.serverKey[serverId])))
		err = errors.New("no logic layer service instance are available")
		s.lock.RUnlock()
		return nil, err
	}
	// 开始轮询获取同一个serverId下的实例
	idx := s.indexMap[serverId] % len(s.serverList[serverId])
	ins = s.serverList[serverId][idx]
	s.lock.RUnlock()

	s.lock.Lock()
	defer s.lock.Unlock()
	// 设置下一次轮询的索引
	s.indexMap[serverId] = (s.indexMap[serverId] + 1) % len(s.serverList[serverId])
	return
}

//Close 关闭服务
func (s *ServiceDiscovery) Close() error {
	return s.cli.Close()
}
