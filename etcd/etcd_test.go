package etcd

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"
)

func TestEtcdDiscover(t *testing.T) {
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

func TestEtcdRegister(t *testing.T) {
	var endpoints = []string{"localhost:2379"}
	ser, err := NewServiceRegister(endpoints, "/server/node1", "localhost:8000", 6, 5)
	if err != nil {
		log.Fatalln(err)
	}
	//监听续租相应chan
	go ser.ListenLeaseRespChan()

	// 监控系统信号，等待 ctrl + c 系统信号通知服务关闭
	c := make(chan os.Signal, 1)
	go func() {
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	}()
	log.Printf("exit %s", <-c)
	ser.Close()

}
