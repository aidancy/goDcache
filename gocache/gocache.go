//Package gocache 负责与外部交互，控制缓存存储和获取的主流程
//
//                           是
// 接收 key --> 检查是否被缓存 -----> 返回缓存值 ⑴
//                 |  否                         是
//                 |-----> 是否应当从远程节点获取 -----> 与远程节点交互 --> 返回缓存值 ⑵
//                             |  否
//                             |-----> 调用`回调函数`，获取值并添加到缓存 --> 返回缓存值 ⑶
package gocache

import (
	"fmt"
	"gocache/singleflight"
	"log"
	"sync"
	pb "gocache/gocachepb"
)

//Getter 由key查找数据并存储
type Getter interface {
	Get(key string) ([]byte, error)
}

//GetterFunc 定义函数类型 GetterFunc
type GetterFunc func(key string) ([]byte, error)

//Get 实现 Getter 接口的 Get 方法
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

//定义一个函数类型 Func，并且实现接口 Aer 的方法，然后在这个方法中调用自己。
//这是 Go 语言中将其他函数（参数返回值定义与 Func 一致）转换为接口 Aer 的常用技巧。

//Group 一个缓存的命名空间
type Group struct {
	name      string
	getter    Getter //缓存未命中时获取源数据的回调(callback)。
	mainCache cache  //并发缓存
	peers     PeerPicker
	loader    *singleflight.Group //确保并发访问只访问一次
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

//NewGroup 实例化 Group，并且将 group 存储在全局变量 groups 中。
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes},
		loader:    &singleflight.Group{},
	}
	groups[name] = g
	return g
}

//GetGroup 回去特定名称的Group
func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

//Get huiqu
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}
	//从 mainCache 中查找缓存，如果存在则返回缓存值。
	if v, ok := g.mainCache.get(key); ok {
		log.Println("[GoCache] hit")
		return v, nil
	}
	//缓存不存在，则调用 load 方法
	return g.load(key)
}

func (g *Group) load(key string) (value ByteView, err error) {
	viewi, err := g.loader.Do(key, func() (interface{}, error) {
		if g.peers != nil {
			//使用 PickPeer() 方法选择节点
			if peer, ok := g.peers.PickPeer(key); ok {
				//若非本机节点，则调用 getFromPeer() 从远程获取
				if value, err := g.getFromPeer(peer, key); err == nil {
					return value, nil
				}
				log.Println("[Gocache] Failed to get from peer", err)
			}
		}
		//若是本机节点或失败
		//getLocally（分布式场景下会调用 getFromPeer 从其他节点获取）
		return g.getLocally(key)
	})
	if err == nil {
		return viewi.(ByteView), nil
	}
	return
}

func (g *Group) getLocally(key string) (ByteView, error) {
	//调用用户回调函数 g.getter.Get() 获取源数据
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}
	value := ByteView{b: cloneBytes(bytes)}
	//将源数据添加到缓存 mainCache 中
	g.populateCache(key, value)
	return value, nil
}

func (g *Group) populateCache(key string, value ByteView) {
	//将源数据添加到缓存 mainCache 中
	g.mainCache.add(key, value)
}

//RegisterPeers 实现了 PeerPicker 接口的 HTTPPool 注入到 Group 中
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}

//getFromPeer PeerGetter 接口的 httpGetter 从访问远程节点，获取缓存值
func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	req := &pb.Request{
		Group: g.name,
		Key: key,
	}
	res := &pb.Response{}
	err := peer.Get(req,res)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: res.Value}, nil
}
