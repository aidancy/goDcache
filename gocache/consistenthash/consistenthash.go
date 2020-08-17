package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

//Hash 依赖注入的方式,允许替换自定义的Hash函数
type Hash func(data []byte) uint32

//Map 一致性哈希算法的主数据结构
type Map struct {
	hash     Hash           //Hash函数
	replicas int            //虚拟节点倍数
	keys     []int          // 哈希环
	hashMap  map[int]string //虚拟节点和真实节点映射表，键是虚拟节点的哈希值，值是真实节点的名称。
}

//New 构造函数，允许自定义虚拟节点倍数和 Hash 函数。
func New(replicas int, fn Hash) *Map {
	m := &Map{
		replicas: replicas,
		hash:     fn,
		hashMap:  make(map[int]string),
	}
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

//Add 添加真实节点,允许传入 0 或 多个真实节点的名称
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		//对应创建 m.replicas 个虚拟节点
		for i := 0; i < m.replicas; i++ {
			//虚拟节点的名称是：编号 + key
			//m.hash() 计算虚拟节点的哈希值
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			//添加到环上
			m.keys = append(m.keys, hash)
			//hashMap 中增加虚拟节点和真实节点的映射关系
			m.hashMap[hash] = key
		}
	}
	//环上的哈希值排序
	sort.Ints(m.keys)
}

//Get 选择节点get
func (m *Map) Get(key string) string {
	if len(m.keys) == 0 {
		return ""
	}
	//计算 key 的哈希值
	hash := int(m.hash([]byte(key)))
	//顺时针找到第一个匹配的虚拟节点的下标
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})
	return m.hashMap[m.keys[idx%len(m.keys)]]
}
