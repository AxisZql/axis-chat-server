# 基于微服务思想的IM系统



## 项目功能简介

* 支持基本的登陆注册功能
* 支持发送一对一的私聊和多对多的群聊消息
* 支持发送文本、图片类型消息
* 支持离线消息
* 支持历史消息记录查看
* 支持实时查看好友在线状态和群聊在线人数的功能
* 支持创建群聊，搜索、添加好友和群聊功能
* 支持用户信息（用户名、头像）自定义功能

## 项目优势阐述

##### 1. 支持高并发场景下的消息发送和投递操作

* 由于本系统采用服务拆分的微服务架构，所以可以通过将对应服务层服务进行水平扩展来提高系统的整体处理能力

* 本系统在消息存放层引入kafka（存放消息数据）和redis list（存放对象状态变更数据）来实现消息的暂时存放，从而达到**流量削峰**的作用，以此来保证即使遇到高并发的消息发送场景也不会出现由于系统处理请求能力不足导致系统全部崩溃的问题。

* 在本系统的connect层使用令牌桶限流算法来处理消息推送的请求，每个消息推送请求在进行之前必须获取令牌后才可以进行将消息推送到对应客户端的操作，从而有效提高系统在高并发场景下的处理能力
  ```go
  type Bucket struct {
  	mutex      sync.RWMutex
  	socketMap  map[int64]*Channel // userid和conn之间的映射
  	GroupNode  map[int64]*GroupNode
  	routineIdx uint64
  	routineNum uint64
  	routines   []chan kafka.Message // 所有要进行消息推送的消息在channel队列中排队
  
  	// statusMsg
  	_routineIdx uint64
  	_routineNum uint64
  	_routines   []chan []byte // 所有要进行消息推送的消息在channel队列中排队
  }
  ```

  * 可以根据目的部署机器的处理性能来指定对应当前connect服务实例下可用的令牌桶数目。

  * 且通过CityHash根据userid（用户id）进行令牌桶的划分，让connect层的对应客户端连接实例均匀地分散到每一个令牌桶中，以此进一步提高系统的处理性能

  * ```go
    func (ws *WsServer) Bucket(userid int64) *Bucket {
    	useridStr := fmt.Sprintf("%d", userid)
    	idx := utils.CityHash32([]byte(useridStr), uint32(len(useridStr))) % ws.bucketNum
    	return ws.Buckets[idx]
    }
    ```

##### 2. 消息投递时序性以及可靠性保证

* 消息投递时许性：
  * 每个对象（用户或者群聊）都在kafka中拥有自身的topic且topic的partition数均为0，以此保证从对应topic读取消息的顺序是严格和消息写入时顺序是一致的
  * 在task层投递每条消息时使用**雪花id**为每条投递的消息打上唯一的id，保证用户后期获取历史消息时消息的时序性
* 消息投递可靠性：
  * 在task层投递消息前必须**获取redis的分布式锁**，且投递消息到connect后成功推送到对应客户端后在connect层**释放redis分布式锁**，**且在认为在task两次获取同一个分布式锁的时间间隔超过30s时则视为前一条消息投递失败，触发消息重投机制**。

## 系统架构

> 本系统一共划分为5层服务「分别为 API层、逻辑(logic)层、消息缓存(kafka)层、消息推送(task)层、连接(connect)层」，**每一层服务都可以单独部署到不同的机器上以实现分布式部署**，其中logic层、task层、kafka层、connect层均可以通过水平扩展的方式提高系统的处理性能。由于每一层服务可以有多个服务实例，故可以有效预防当由于其中一层服务其中一实例的异常崩溃而造成整个系统瘫痪的事故发生。

* 系统架构图

  <img src="https://static.axiszql.com/articles/3b712bddd23a557d5f30f58ed6bf949f.png" alt="image-20220710165434587" style="zoom:67%;" />



* **启用RPC服务的服务层采用etcd进行服务的注册和发现功能：**
  <img src="https://static.axiszql.com/articles/612ed03212beba247034f29ed5c715e8.png" alt="image-20220710165037011" style="zoom:50%;" />

  > 其中logic层和connect层的RPC服务都注册到etcd中，然后etcd使用实时检测这些服务都在线情况。如果其他层的服务需要调用这些RPC服务时，它们必须通过访问etcd来获取对应服务层上的服务实例的地址，以此来达到服务注册和发现的功能。



* **connect层单个server（对应一个serverId）内部结构逻辑图：**

  ![image-20220710193217878](https://static.axiszql.com/articles/c2b9d120724a579d8e9b5513c08f78e9.png)

  > 由于本系统中每一个用户至少存在于一个群聊当中，所以每一个GroupNodes节点下维护一个双向链表来记录当前serverId当前群聊中所有在线用户的WebSocket连接实例，以此简化群发消息推送操作以及对应对象（群聊或者用户）下线时的记录销毁操作
  >
  > 温馨提示：以上图中的channel对应一个用户连接实例而并非Golang中的chan类型

  * 数据结构定义：

    ```go
    type Channel struct {
    	BroadcastMsg    chan kafka.Message 
    	BroadcastStatus chan []byte        
    	Userid          int64
    	Conn            *websocket.Conn
    	Next            *Channel 
    	Prev            *Channel
    	GroupNodes      []*GroupNode // 
    }
    ```

    > 部分字段含义解析：
    >
    > BroadcastMsg：要推送给当前连接对应用户的消息数据缓冲channel
    >
    > BoadcastStatus：  状态消息数据缓冲Channel
    >
    > Conne：WebSocket连接实例
    >
    > GroupNodes：记录当前Channel（用户连接实例）持有的所有GroupNode，方便后期用户下线的时候，能快速从对应的GroupNode的双向链表中删除对应的Channel（因为一位用户可能有多个群聊）