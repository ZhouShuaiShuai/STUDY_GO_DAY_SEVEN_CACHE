package lru

import (
	"container/list"
	"fmt"
)

/**
在这里我们直接使用 Go 语言标准库实现的双向链表list.List。
字典的定义是 map[string]*list.Element，键是字符串，值是双向链表中对应节点的指针。
maxBytes 是允许使用的最大内存，nbytes 是当前已使用的内存，OnEvicted 是某条记录被移除时的回调函数，可以为 nil。
键值对 entry 是双向链表节点的数据类型，在链表中仍保存每个值对应的 key 的好处在于，淘汰队首节点时，需要用 key 从字典中删除对应的映射。
为了通用性，我们允许值是实现了 Value 接口的任意类型，该接口只包含了一个方法 Len() int，用于返回值所占用的内存大小。
*/

type Cache struct {
	maxbytes  int64      //允许使用的最大内存
	nbytes    int64      //当前已使用的内存
	ll        *list.List //链表
	cache     map[string]*list.Element
	OnEvicted func(key string, value Value) //某条记录被移除时的回调函数，可以为 nil。
}

type entry struct {
	key   string
	value Value
}

type Value interface {
	Len() int
}

/*
方便测试，我们实现 Len() 用来获取添加了多少条数据。
*/
func (c *Cache) Len() int {
	return c.ll.Len()
}

//方便实例化 Cache，实现 New() 函数
func New(maxbytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxbytes:  maxbytes,
		OnEvicted: onEvicted,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
	}
}

/*
查找:
第一步是从字典中找到对应的双向链表的节点.
第二步，将该节点移动到队尾.

如果键对应的链表节点存在，则将对应节点移动到队尾，并返回查找到的值。
c.ll.MoveToFront(ele)，即将链表中的节点 ele 移动到队尾（双向链表作为队列，队首队尾是相对的，在这里约定 front 为队尾）
*/

//* 号用于指定变量是作为一个指针
func (c *Cache) Get(key string) (value Value, ok bool) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		return kv.value, true
	}
	return
}

/*
删除：
这里的删除，实际上是缓存淘汰。即移除最近最少访问的节点（队首）

c.ll.Back() 取到队首节点，从链表中删除。
delete(c.cache, kv.key)，从字典中 c.cache 删除该节点的映射关系。
更新当前所用的内存 c.nbytes。
如果回调函数 OnEvicted 不为 nil，则调用回调函数。
*/
func (c *Cache) RemoveOldest() {
	ele := c.ll.Back()
	if ele != nil {
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		delete(c.cache, kv.key)
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len())
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

/*
新增/修改

如果键存在，则更新对应节点的值，并将该节点移到队尾。
不存在则是新增场景，首先队尾添加新节点 &entry{key, value}, 并字典中添加 key 和节点的映射关系。
更新 c.nbytes，如果超过了设定的最大值 c.maxBytes，则移除最少访问的节点。
*/
func (c *Cache) Add(key string, value Value) {
	fmt.Println("ADD FUNC")
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else {
		ele := c.ll.PushFront(&entry{key, value})
		c.cache[key] = ele
		c.nbytes += int64(len(key)) + int64(value.Len())
	}
	for c.maxbytes != 0 && c.maxbytes < c.nbytes {
		c.RemoveOldest()
	}
}
