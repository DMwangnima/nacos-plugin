package selector

import (
	"errors"
	microErr "github.com/asim/go-micro/v3/errors"
	"github.com/asim/go-micro/v3/registry"
	"github.com/asim/go-micro/v3/registry/cache"
	"github.com/asim/go-micro/v3/selector"
	"sync"
	"time"
)

const (
	REFRESH_INTERVAL = 30 * time.Second // 最好给缓存设置一个随机的过期时间，否则会在某个时间节点对nacos造成较大的访问压力
)

type nacosSelector struct {
	options selector.Options
	cache   cache.Cache
	exit    chan struct{}

	mu sync.RWMutex
	// 各服务对应的next func
	nextFuncs map[string]selector.Next
	// 针对用户使用的next func
	userNextFuncs map[string]selector.Next
	markMap       map[string]chan string
	markFinMap    map[string]chan struct{}
}

func (n *nacosSelector) Init(opts ...selector.Option) error {
	for _, opt := range opts {
		opt(&n.options)
	}
	return nil
}

func (n *nacosSelector) Options() selector.Options {
	return n.options
}

// 定期请求缓存
func (n *nacosSelector) watch(service string, update chan []*registry.Service) {
	updateFunc := func() {
		services, err := n.cache.GetService(service)
		if err != nil {
			return
		}
		update <- services
	}
	// 第一次调用时，直接请求一次缓存
	updateFunc()
	ticker := time.NewTicker(REFRESH_INTERVAL)
	defer ticker.Stop()

	for {
		select {
		case <-n.exit:
			return
		case <-ticker.C:
			updateFunc()
		}
	}
}

// 更新next函数
func (n *nacosSelector) changeNext(service string, update chan []*registry.Service, onceFinish chan struct{}) {
	// 标识是否第一次运行
	onceFlag := true
	var curServices []*registry.Service
	n.mu.RLock()
	mark := n.markMap[service]
	markFin := n.markFinMap[service]
	n.mu.RUnlock()
	changeNext := func() {
		n.mu.Lock()
		n.nextFuncs[service] = n.options.Strategy(curServices)
		n.mu.Unlock()
	}
	for {
		select {
		case <-n.exit:
			return
		case addr := <-mark:
			// 删除该标记节点
			for _, s := range curServices {
				for j, n := range s.Nodes {
					if addr == n.Address {
						s.Nodes = append(s.Nodes[:j], s.Nodes[j+1:]...)
						break
					}
				}
			}
			changeNext()
			markFin <- struct{}{}
		case newService := <-update:
			curServices = newService
			changeNext()
			if onceFlag {
				onceFlag = false
				onceFinish <- struct{}{}
			}
		}
	}
}

func (n *nacosSelector) Select(service string, opts ...selector.SelectOption) (selector.Next, error) {
	sOpts := selector.SelectOptions{
		Strategy: n.options.Strategy,
	}
	for _, opt := range opts {
		opt(&sOpts)
	}

	n.mu.RLock()
	if funct, ok := n.userNextFuncs[service]; ok {
		n.mu.RUnlock()
		return funct, nil
	}
	n.mu.RUnlock()

	n.mu.Lock()
	n.markMap[service] = make(chan string)
	n.markFinMap[service] = make(chan struct{})
	n.mu.Unlock()

	update := make(chan []*registry.Service)
	go n.watch(service, update)
	onceFinish := make(chan struct{})
	go n.changeNext(service, update, onceFinish)

	next := func() (*registry.Node, error) {
		n.mu.RLock()
		next := n.nextFuncs[service]
		n.mu.RUnlock()
		return next()
	}
	<-onceFinish
	n.mu.Lock()
	n.userNextFuncs[service] = next
	n.mu.Unlock()
	return next, nil
}

func (n *nacosSelector) Mark(service string, node *registry.Node, err error) {
	// todo: 对err进行分类
	if err == nil || service == "" || node == nil {
		return
	}
	Err, ok := err.(*microErr.Error)
	if !ok {
		return
	}
	// 还需要确定error的具体含义
	switch Err.Code {
	case 4:
	case 14:
	default:
		return
	}
	n.mu.RLock()
	mark := n.markMap[service]
	markFin := n.markFinMap[service]
	n.mu.RUnlock()
	mark <- node.Address
	<-markFin
}

// 暂不支持reset
func (n *nacosSelector) Reset(service string) {
	return
}

func (n *nacosSelector) Close() error {
	select {
	case <-n.exit:
		return errors.New("nacos selector has been closed")
	default:
		close(n.exit)
		return nil
	}
}

func (n *nacosSelector) String() string {
	return "nacos"
}

func (n *nacosSelector) newCache() cache.Cache {
	return cache.New(n.options.Registry)
}

func NewSelector(opts ...selector.Option) selector.Selector {
	options := selector.Options{
		Strategy: selector.Random,
	}
	for _, opt := range opts {
		opt(&options)
	}
	if options.Registry == nil {
		panic("nacos selector: missing registry options")
	}
	s := &nacosSelector{
		options:       options,
		exit:          make(chan struct{}),
		markMap:       make(map[string]chan string),
		nextFuncs:     make(map[string]selector.Next),
		userNextFuncs: make(map[string]selector.Next),
		markFinMap:    make(map[string]chan struct{}),
	}
	s.cache = s.newCache()
	return s
}
