package etcd

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/coreos/etcd/mvcc/mvccpb"
	"go.etcd.io/etcd/clientv3"
)

//ServiceDiscovery 服务发现
type ServiceDiscovery struct {
	cli        *clientv3.Client  //etcd client
	serverList map[string]string //注册到etcd的所有服务的列表，建立对应path和旗下所有服务的映射
	lock       sync.RWMutex
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
		serverList: make(map[string]string),
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
	s.lock.Lock()
	defer s.lock.Unlock()
	s.serverList[key] = val
	log.Println("put key :", key, "val:", val)
}

//DelServiceList 删除服务地址
func (s *ServiceDiscovery) DelServiceList(key string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.serverList, key)
	log.Println("del key:", key)
}

//GetServices 获取注册到etcd所有path下的服务地址
func (s *ServiceDiscovery) GetServices() []string {
	s.lock.RLock()
	defer s.lock.RUnlock()
	addrs := make([]string, 0, len(s.serverList))

	for _, v := range s.serverList {
		addrs = append(addrs, v)
	}
	return addrs
}

//Close 关闭服务
func (s *ServiceDiscovery) Close() error {
	return s.cli.Close()
}

func main() {
	var endpoints = []string{"localhost:2379"}
	ser := NewServiceDiscovery(endpoints)
	defer ser.Close()

	err := ser.WatchService("/server/")
	if err != nil {
		log.Fatal(err)
	}

	// 监控系统信号，等待 ctrl + c 系统信号通知服务关闭
	c := make(chan os.Signal, 1)
	go func() {
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	}()

	for {
		select {
		case <-time.Tick(10 * time.Second):
			log.Println(ser.GetServices())
		case <-c:
			log.Println("server discovery exit")
			return
		}
	}
}
