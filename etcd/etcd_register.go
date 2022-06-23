package etcd

import (
	"axisChat/utils/zlog"
	"context"
	"fmt"
	"time"

	"go.etcd.io/etcd/client/v3"
)

//ServiceRegister 创建租约注册服务
type ServiceRegister struct {
	cli     *clientv3.Client //etcd v3 client
	leaseID clientv3.LeaseID //租约ID
	//租约keepalive相应chan
	keepAliveChan <-chan *clientv3.LeaseKeepAliveResponse
	key           string //key
	val           string //value
}

//NewServiceRegister 新建注册服务
func NewServiceRegister(endpoints []string, key, val string, lease int64, dailTimeout int) (*ServiceRegister, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: time.Duration(dailTimeout) * time.Second,
	})
	if err != nil {
		return nil, err
	}

	ser := &ServiceRegister{
		cli: cli,
		key: key,
		val: val,
	}

	//申请租约设置时间keepalive，TODO:lease 为租约的ttl
	if err := ser.putKeyWithLease(lease); err != nil {
		return nil, err
	}

	return ser, nil
}

//设置租约
func (s *ServiceRegister) putKeyWithLease(lease int64) error {
	//创建一个新的租约，并设置ttl时间
	resp, err := s.cli.Grant(context.Background(), lease)
	if err != nil {
		return err
	}

	//注册服务并绑定租约
	_, err = s.cli.Put(context.Background(), s.key, s.val, clientv3.WithLease(resp.ID))
	if err != nil {
		return err
	}
	//设置续租 定期发送需求请求
	//KeepAlive使给定的租约永远有效。 如果发布到通道的keepalive响应没有立即被使用，
	// 则租约客户端将至少每秒钟继续向etcd服务器发送保持活动请求，直到获取最新的响应为止。
	//etcd client会自动发送ttl到etcd server，从而保证该租约一直有效
	leaseRespChan, err := s.cli.KeepAlive(context.Background(), resp.ID)
	if err != nil {
		return err
	}

	s.leaseID = resp.ID
	zlog.Info(fmt.Sprintf("%v", s.leaseID))
	s.keepAliveChan = leaseRespChan
	zlog.Info(fmt.Sprintf("Put key:%s  val:%s  success!", s.key, s.val))
	return nil
}

//ListenLeaseRespChan 监听 续租情况
func (s *ServiceRegister) ListenLeaseRespChan() {
	defer func() {
		zlog.Info("关闭续租")
	}()
	zlog.Info("开始续约")
	for {
		select {
		case _, ok := <-s.keepAliveChan:
			if !ok {
				zlog.Error("服务续租关闭了")
				return
			}
		}
	}
}

// Close 注销服务
func (s *ServiceRegister) Close() error {
	//撤销租约
	if _, err := s.cli.Revoke(context.Background(), s.leaseID); err != nil {
		return err
	}
	zlog.Info("撤销租约")
	return s.cli.Close()
}
