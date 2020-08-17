//Package lru lru缓存淘汰策略
package lru

import "container/list"

// Cache LRUcache，并发访问不安全
type Cache struct {
	maxBytes  int64	//允许使用的最大内存
	nbytes    int64	//当前已使用的内存
	ll        *list.List //go标准库中的双向链表
	cache     map[string]*list.Element	//键是字符串，值是双向链表中对应节点的指针
	OnEvicted func(key string, value Value)	//某条记录被移除时的回调函数
}

// entry 双向链表节点
type entry struct {
	key   string
	value Value
}

//Value 其中的len()返回Value占用多少bytes
type Value interface {
	Len() int //返回值所占用的内存大小
}

//New 实例化Cache
func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes: maxBytes,
		ll: list.New(),
		cache: make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

//Get 查找
func (c *Cache) Get(key string) (value Value, ok bool) {
	//从字典中找到对应的双向链表的节点
	if ele, ok := c.cache[key]; ok {
		//将该节点移动到队尾
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry) 
		return kv.value, true
	}
	return
}

//RemoveOldest 缓存淘汰，移除最近最少访问节点
func (c *Cache) RemoveOldest()  {
	//取队首节点，从链表中删除
	ele := c.ll.Back()
	if ele != nil {
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		//删除字典里的映射
		delete(c.cache, kv.key)
		//更新所用内存大小
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len())
		//如果回调函数不为空，调用回调函数
		if c.OnEvicted != nil{
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

//Add 新增和修改
func (c *Cache) Add(key string, value Value)  {
	//如果键存在，则更新对应节点的值，并将该节点移到队尾。
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else {
		//新增
		//队尾添加新节点
		ele := c.ll.PushFront(&entry{key, value})
		//字典中添加映射关系
		c.cache[key] = ele
		//更新 c.nbytes
		c.nbytes += int64(len(key)) + int64(value.Len())
	}
	//c.nbytes，如果超过了设定的最大值 c.maxBytes，则移除最少访问的节点。
	for c.maxBytes != 0 && c.maxBytes < c.nbytes {
		c.RemoveOldest()
	}
}

//Len 回去添加了多少条数据
func (c *Cache) Len() int {
	return c.ll.Len()
}