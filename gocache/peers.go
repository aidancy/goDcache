package gocache

// 使用一致性哈希选择节点        是                                    是
//     |-----> 是否是远程节点 -----> HTTP 客户端访问远程节点 --> 成功？-----> 服务端返回返回值
//                     |  否                                    ↓  否
//                     |----------------------------> 回退到本地节点处理。

import pb "gocache/gocachepb"

//PeerPicker 选择相应节点
type PeerPicker interface {
	//PickPeer() 方法用于根据传入的 key 选择相应节点 PeerGetter。
	PickPeer(key string) (peer PeerGetter, ok bool)
}

//PeerGetter HTTP客户端
type PeerGetter interface {
	//用于从对应 group 查找缓存值
	Get(in *pb.Request, out *pb.Response) error
}
